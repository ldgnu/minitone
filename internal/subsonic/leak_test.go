package subsonic

import (
	"strings"
	"testing"
)

// El token de autenticación viaja en la query string. ¿Acaba en el mensaje de
// error que minitone muestra en pantalla?
func TestAuthTokenDoesNotLeakIntoErrors(t *testing.T) {
	// Servidor que no existe: http.Client devuelve *url.Error con la URL entera.
	c := NewClient("http://127.0.0.1:1", "alice", "s3cr3t-pass")

	err := c.Ping()
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()

	for _, secret := range []string{"s3cr3t-pass", c.token, c.salt} {
		if secret == "" {
			continue
		}
		if strings.Contains(msg, secret) {
			t.Errorf("LEAK: the error message exposes %q\n  err = %v", secret, msg)
		}
	}
	if strings.Contains(msg, "alice") {
		t.Errorf("LEAK: the error message exposes the username\n  err = %v", msg)
	}
}
