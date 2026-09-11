package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ols-cli/ols/internal/config"
)

func TestFetchWordPressSalts(t *testing.T) {
	salts, err := FetchWordPressSalts()
	if err != nil {
		t.Fatalf("FetchWordPressSalts failed: %v", err)
	}

	if !strings.Contains(salts, "AUTH_KEY") || !strings.Contains(salts, "NONCE_SALT") {
		t.Errorf("Returned salts missing expected keys: %s", salts)
	}
}

func TestRegenerateSalts(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "test-sec.com"
	siteDir := filepath.Join(tmpDir, "sites", domain)
	htmlDir := filepath.Join(siteDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	initialConfig := `<?php
define( 'DB_NAME', 'wp_test' );
// Authentication Unique Keys and Salts
define( 'AUTH_KEY',         'old-key-1' );
define( 'SECURE_AUTH_KEY',  'old-key-2' );
define( 'LOGGED_IN_KEY',    'old-key-3' );
define( 'NONCE_KEY',        'old-key-4' );
define( 'AUTH_SALT',        'old-key-5' );
define( 'SECURE_AUTH_SALT', 'old-key-6' );
define( 'LOGGED_IN_SALT',   'old-key-7' );
define( 'NONCE_SALT',       'old-key-8' );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
`
	wpConfigPath := filepath.Join(htmlDir, "wp-config.php")
	if err := os.WriteFile(wpConfigPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := &Manager{
		cfg: &config.Config{
			SystemDir: tmpDir,
		},
	}

	ok, err := mgr.RegenerateSalts(domain)
	if err != nil || !ok {
		t.Fatalf("RegenerateSalts failed: %v", err)
	}

	updatedBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	updatedContent := string(updatedBytes)
	if strings.Contains(updatedContent, "old-key-1") {
		t.Errorf("Old salts were not replaced properly: %s", updatedContent)
	}
	if !strings.Contains(updatedContent, "AUTH_KEY") || !strings.Contains(updatedContent, "NONCE_SALT") {
		t.Errorf("New salts missing in wp-config.php: %s", updatedContent)
	}
}

func TestGetSitePHPBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	mgr := NewManager(cfg)

	// Case 1: Site with PHP 8.3 compose
	siteDir83 := filepath.Join(tmpDir, "sites", "site83.com")
	_ = os.MkdirAll(siteDir83, 0755)
	_ = os.WriteFile(filepath.Join(siteDir83, "docker-compose.yml"), []byte("services:\n  ols:\n    image: litespeedtech/openlitespeed:1.8.2-lsphp83\n"), 0644)
	if bin := mgr.GetSitePHPBinary("site83.com"); bin != "/usr/local/lsws/lsphp83/bin/php" {
		t.Errorf("expected /usr/local/lsws/lsphp83/bin/php, got %s", bin)
	}

	// Case 2: Default fallback to 8.2
	if bin := mgr.GetSitePHPBinary("nonexistent.com"); bin != "/usr/local/lsws/lsphp82/bin/php" {
		t.Errorf("expected default /usr/local/lsws/lsphp82/bin/php, got %s", bin)
	}
}

func TestReplaceSaltsInWPConfig_DollarSignAndSpecialChars(t *testing.T) {
	initial := `<?php
define( 'DB_NAME', 'wp_test' );
// Authentication Unique Keys and Salts
define( 'AUTH_KEY',         'old1' );
define( 'SECURE_AUTH_KEY',  'old2' );
define( 'LOGGED_IN_KEY',    'old3' );
define( 'NONCE_KEY',        'old4' );
define( 'AUTH_SALT',        'old5' );
define( 'SECURE_AUTH_SALT', 'old6' );
define( 'LOGGED_IN_SALT',   'old7' );
define( 'NONCE_SALT',       'old8' );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
`

	// Salts containing $0, $1, $2, ${foo}, $`, etc. which would break ReplaceAllString
	complexSalts := `define('AUTH_KEY',         'a$0b$1c$2d$name');
define('SECURE_AUTH_KEY',  'test${foo}bar');
define('LOGGED_IN_KEY',    'val$$double');
define('NONCE_KEY',        'key$');
define('AUTH_SALT',        'salt$7end');
define('SECURE_AUTH_SALT', 'salt}curly');
define('LOGGED_IN_SALT',   'salt{sk<#%');
define('NONCE_SALT',       'final$1$2salt');`

	result := ReplaceSaltsInWPConfig(initial, complexSalts)

	// Verify all special sequences are preserved verbatim
	expectedSubstrings := []string{
		"a$0b$1c$2d$name",
		"test${foo}bar",
		"val$$double",
		"salt$7end",
		"salt}curly",
		"salt{sk<#%",
		"final$1$2salt",
		"if ( ! defined( 'ABSPATH' ) ) {",
	}

	for _, exp := range expectedSubstrings {
		if !strings.Contains(result, exp) {
			t.Errorf("expected result to contain %q, but got:\n%s", exp, result)
		}
	}

	// Verify old salts are completely removed
	if strings.Contains(result, "old1") || strings.Contains(result, "old8") {
		t.Errorf("old salts still present in result:\n%s", result)
	}

	// Test regenerating a second time (with previously generated comment)
	secondSalts := `define('AUTH_KEY',         'second-round-key');
define('NONCE_SALT',       'second-round-nonce');`
	secondResult := ReplaceSaltsInWPConfig(result, secondSalts)

	if !strings.Contains(secondResult, "second-round-key") {
		t.Errorf("second regeneration failed to replace salts")
	}
	if strings.Contains(secondResult, "a$0b$1c$2d$name") {
		t.Errorf("first round salts still present after second round")
	}
}

func TestReplaceSaltsInWPConfig_CorruptedOrphanLine(t *testing.T) {
	// Exact scenario from user's hb88.dental with orphan line 48 and duplicate salts on lines 49-52
	corrupted := "<?php\ndefine('DB_NAME', 'hb88');\n" +
		"define('AUTH_KEY',         '{?3`_Hei)fL.};tU+9aS-wc}Ju1ys(g(<T|8cRa76$:`BX:B;#.7:XpJ3c(cF/KY');\n" +
		"define('SECURE_AUTH_KEY',  '*LJP%E9gc.d`D_kU@&_Dcp<d=,|SWlV1E`fwG-ov87)AgBsg-cz:Lztvfo]W{:.D');\n" +
		"define('LOGGED_IN_KEY',    'B[+4KBj<eY]};<M|rGJ0rm8/RJ[6*u#7jvyL$03 &GQT;[EekEau_DqN&#J/6S|!');\n" +
		"define('NONCE_KEY',        'VZ!|~_V+[|yg#@*n^yj. 4iPx>??_6-s<dkD-~<bKtkxu>1m Z30~6D%3TY?|s$s');\n" +
		"define('AUTH_SALT',        '^1`FYjA3G{5|>jY%|4/x?Ab:><S+=$X.;C/e;kk+IljDjtcHKswP$`iSM6$V;P;=');\n" +
		"define('SECURE_AUTH_SALT', '4v?KPiwC+d;C6:PB5ncn:0>j?V08+*H<[>hXsisqtxk{+|W:-@e)L3ed&?qpGDsd');\n" +
		"define('LOGGED_IN_SALT',   'A&}gMw;:|^-5%J`G[G6K$S|.!MfZy}Si0oZ=IF?t>/!M#>EC6@pSkyBrdF-0g]n5');\n" +
		"define('NONCE_SALT',       '*,Mw90ydRo[-AVotTklSXd}.lXFMCxZS&^S!?jE^r]&7GJNgd$LFV{Kd4Zz+#j1!');\n" +
		"}T-Pw}*<YJTdpO.UeRcx');\n" +
		"define('AUTH_SALT',        'By]j>`jJ=PY.[/EhAxizX:|r uQj#[R16<=[eDoRFmb+IaZQ&zj@D<hcfu#');\n" +
		"define('SECURE_AUTH_SALT', 'WHp+gqvhSi.)8S5FxA.o5zSB[BoAn!S-^`YOHN)9+tALmytQAaqgLN?GA#6BKqFD');\n" +
		"define('LOGGED_IN_SALT',   'rOI0f%*bG@YUKU .6c[zNV_VwvtB`o#-)DF5|zQVT&J|60)MyO=>P<X@ee^.xJm&');\n" +
		"define('NONCE_SALT',       '<rAqN[QeQF|Sw6d7Bq)aZ8yVM+7;-/[@+@u%d0Pa$(_@Bi]S+?of&>+5uv+9T@rz');\n\n" +
		"$table_prefix = 'wp_';\n" +
		"if ( ! defined( 'ABSPATH' ) ) {\n\tdefine( 'ABSPATH', __DIR__ . '/' );\n}\n"

	cleanSalts := `define('AUTH_KEY',         'clean_k1');
define('SECURE_AUTH_KEY',  'clean_k2');
define('LOGGED_IN_KEY',    'clean_k3');
define('NONCE_KEY',        'clean_k4');
define('AUTH_SALT',        'clean_s1');
define('SECURE_AUTH_SALT', 'clean_s2');
define('LOGGED_IN_SALT',   'clean_s3');
define('NONCE_SALT',       'clean_s4');`

	result := ReplaceSaltsInWPConfig(corrupted, cleanSalts)

	// Ensure orphan line and old duplicate salts are wiped clean
	if strings.Contains(result, "}T-Pw}") {
		t.Errorf("corrupted orphan line '}T-Pw}' was not cleaned up!")
	}
	if strings.Contains(result, "By]j>`jJ") || strings.Contains(result, "WHp+gqvh") {
		t.Errorf("duplicate old salts were not cleaned up!")
	}
	if strings.Count(result, "NONCE_SALT") != 1 {
		t.Errorf("expected exactly 1 NONCE_SALT, got %d", strings.Count(result, "NONCE_SALT"))
	}
	if !strings.Contains(result, "clean_s4") {
		t.Errorf("expected clean salts in output")
	}
	if !strings.Contains(result, "$table_prefix = 'wp_';") {
		t.Errorf("table prefix missing or corrupted")
	}
}

func TestParseResetPasswordOutput(t *testing.T) {
	// Case 1: Pure SUCCESS
	res, err := ParseResetPasswordOutput("SUCCESS:admin", "Secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res, "User: admin") || !strings.Contains(res, "Secret123") {
		t.Errorf("unexpected format: %s", res)
	}

	// Case 2: SUCCESS with leading PHP warning
	outWithWarning := "PHP Warning: Cannot modify header\nSUCCESS:superadmin\n"
	res, err = ParseResetPasswordOutput(outWithWarning, "Pass@456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res, "User: superadmin") || !strings.Contains(res, "Pass@456") {
		t.Errorf("unexpected format: %s", res)
	}

	// Case 3: ERROR from WordPress
	outErr := "PHP Notice: bla\nERROR: Không tìm thấy tài khoản quản trị viên"
	_, err = ParseResetPasswordOutput(outErr, "pass")
	if err == nil || !strings.Contains(err.Error(), "Không tìm thấy tài khoản") {
		t.Errorf("expected error containing 'Không tìm thấy tài khoản', got %v", err)
	}
}


