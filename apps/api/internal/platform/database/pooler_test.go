package database

import "testing"

func TestIsTransactionPooler(t *testing.T) {
	// Getting this wrong in either direction is expensive: miss a pooler and
	// production fails intermittently with 08P01; see one that is not there
	// and every query gives up its prepared statement for nothing.
	pooled := map[string]string{
		"neon":              "ep-flat-tooth-b26zn32a-pooler.c-6.eu-central-1.aws.neon.tech",
		"pgbouncer in host": "db-pgbouncer.example.com",
	}
	for name, host := range pooled {
		if !isTransactionPooler("postgres://u:p@"+host+"/db", host) {
			t.Errorf("%s: %q was not recognised as pooled", name, host)
		}
	}

	direct := map[string]string{
		"neon direct": "ep-flat-tooth-b26zn32a.c-6.eu-central-1.aws.neon.tech",
		"plain":       "db.example.com",
		"localhost":   "localhost",
	}
	for name, host := range direct {
		if isTransactionPooler("postgres://u:p@"+host+"/db", host) {
			t.Errorf("%s: %q was treated as pooled", name, host)
		}
	}

	// The parameter poolers conventionally take, on a host that says nothing.
	if !isTransactionPooler("postgres://u:p@db.example.com/db?pgbouncer=true", "db.example.com") {
		t.Error("pgbouncer=true was ignored")
	}
}
