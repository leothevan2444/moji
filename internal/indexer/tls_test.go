package indexer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"math/big"
	"net/url"
	"testing"
	"time"
)

func TestDeclaredCertificateTrustIsScopedAndStillRejectsOtherCertificates(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"site.invalid"}, NotBefore: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, private)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(der)
	network := NewNetwork(context.Background())
	defer network.Close()
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, []string{hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	target, _ := url.Parse("https://site.invalid/")
	transport, err := session.transport.forOrigin(target)
	if err != nil {
		t.Fatal(err)
	}
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{certificate}}
	if err := transport.TLSClientConfig.VerifyConnection(state); err != nil {
		t.Fatal("declared certificate not accepted", err)
	}
	other, _ := url.Parse("https://cdn.invalid/")
	cdn, err := session.transport.forOrigin(other)
	if err != nil {
		t.Fatal(err)
	}
	if cdn.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("site certificate exception reached CDN")
	}
	unmatched, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, []string{"0000000000000000000000000000000000000000"})
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := unmatched.transport.forOrigin(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := configuration.TLSClientConfig.VerifyConnection(state); err == nil {
		t.Fatal("unlisted expired certificate accepted")
	}
}
