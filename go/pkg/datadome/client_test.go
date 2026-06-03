package datadome_test

import (
	"testing"

	"github.com/CircuitSavage/datadome-solver/internal/builder"
	ddcrypto "github.com/CircuitSavage/datadome-solver/internal/crypto"
	"github.com/CircuitSavage/datadome-solver/pkg/datadome"
)

func TestNewRequiresSiteAndKey(t *testing.T) {
	_, err := datadome.New("")
	if err == nil {
		t.Fatal("expected error for empty site")
	}
	_, err = datadome.New("https://example.com/")
	if err == nil {
		t.Fatal("expected error without DDJSKey")
	}
}

func TestNewWithProxy(t *testing.T) {
	c, err := datadome.New(
		"https://example.com/",
		datadome.WithDDJSKey("0000000000000000000000000000000"),
		datadome.WithProxy("http://127.0.0.1:8080"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.ProxyURL != "http://127.0.0.1:8080" {
		t.Fatalf("proxy = %q", c.ProxyURL)
	}
}

func TestBuildAndEncrypt(t *testing.T) {
	c, err := datadome.New(
		"https://example.com/",
		datadome.WithDDJSKey("0000000000000000000000000000000"),
	)
	if err != nil {
		t.Fatal(err)
	}
	signals := c.BuildPayload(nil, 1)
	if len(signals) < 100 {
		t.Fatalf("expected many signals, got %d", len(signals))
	}
	jspl, err := c.EncryptJSPL(signals)
	if err != nil {
		t.Fatal(err)
	}
	if len(jspl) < 100 {
		t.Fatalf("jspl too short: %d", len(jspl))
	}
}

func TestEncryptDeterministic(t *testing.T) {
	signals := []ddcrypto.Signal{{Key: "log2", Value: "gl,tzp"}}
	ts := int64(1700000000000)
	key := "0000000000000000000000000000000"

	a, err := ddcrypto.Encrypt(signals, key, "", &ts)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ddcrypto.Encrypt(signals, key, "", &ts)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("same inputs should yield same jspl")
	}
}

func TestBuilderUsesSiteURL(t *testing.T) {
	signals := builder.BuildPayload(builder.Options{
		Profile: "chrome_win10",
		URL:     "https://example.com/",
		BPC:     1,
	})
	var ua string
	for _, s := range signals {
		if s.Key == "ua" {
			ua, _ = s.Value.(string)
		}
	}
	if ua == "" {
		t.Fatal("missing ua signal")
	}
}
