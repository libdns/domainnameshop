package domainnameshop_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libdns/domainnameshop"
	"github.com/libdns/libdns"
)

var (
	envToken  = ""
	envSecret = ""
	envZone   = ""
	ttl       = time.Duration(120 * time.Second)
)

type testRecordsCleanup = func()

func setupTestRecords(t *testing.T, p *domainnameshop.Provider) ([]libdns.Record, testRecordsCleanup) {
	testRecords := []libdns.Record{
		libdns.RR{
			Type: "TXT",
			Name: "test1",
			Data: "test1",
			TTL:  ttl,
		}, libdns.RR{
			Type: "TXT",
			Name: "test2",
			Data: "test2",
			TTL:  ttl,
		}, libdns.RR{
			Type: "TXT",
			Name: "test3",
			Data: "test3",
			TTL:  ttl,
		},
	}

	records, err := p.AppendRecords(context.TODO(), envZone, testRecords)
	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, p, records)
	}
}

func cleanupRecords(t *testing.T, p *domainnameshop.Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(context.TODO(), envZone, r)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestMain(m *testing.M) {
	envToken = os.Getenv("LIBDNS_DOMAINNAMESHOP_TEST_TOKEN")
	envSecret = os.Getenv("LIBDNS_DOMAINNAMESHOP_TEST_SECRET")
	envZone = os.Getenv("LIBDNS_DOMAINNAMESHOP_TEST_ZONE")

	if len(envToken) == 0 || len(envSecret) == 0 || len(envZone) == 0 {
		fmt.Println(`Please notice that this test runs agains the public Domainname.shop DNS Api, so you sould
never run the test with a zone, used in production.
To run this test, you have to specify 'LIBDNS_DOMAINNAMESHOP_TEST_TOKEN', 'LIBDNS_DOMAINNAMESHOP_TEST_SECRET' and 'LIBDNS_DOMAINNAMESHOP_TEST_ZONE'.
Example: "LIBDNS_HETZNER_TEST_TOKEN="123" LIBDNS_DOMAINNAMESHOP_TEST_SECRET="123" LIBDNS_HETZNER_TEST_ZONE="my-domain.com" go test ./... -v`)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func Test_AppendRecords(t *testing.T) {
	p := &domainnameshop.Provider{
		APIToken:  envToken,
		APISecret: envSecret,
	}

	testCases := []struct {
		records  []libdns.Record
		expected []libdns.Record
	}{
		{
			// multiple records
			records: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "test_1", Data: "test_1", TTL: ttl},
				libdns.RR{Type: "TXT", Name: "test_2", Data: "test_2", TTL: ttl},
				libdns.RR{Type: "TXT", Name: "test_3", Data: "test_3", TTL: ttl},
			},
			expected: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "test_1", Data: "test_1", TTL: ttl},
				libdns.RR{Type: "TXT", Name: "test_2", Data: "test_2", TTL: ttl},
				libdns.RR{Type: "TXT", Name: "test_3", Data: "test_3", TTL: ttl},
			},
		},
		{
			// relative name
			records: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "123.test", Data: "123", TTL: ttl},
			},
			expected: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "123.test", Data: "123", TTL: ttl},
			},
		},
		{
			// (fqdn) sans trailing dot
			records: []libdns.Record{
				libdns.RR{Type: "TXT", Name: fmt.Sprintf("123.test.%s", strings.TrimSuffix(envZone, ".")), Data: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "123.test", Data: "test", TTL: ttl},
			},
		},
		{
			// fqdn with trailing dot
			records: []libdns.Record{
				libdns.RR{Type: "TXT", Name: fmt.Sprintf("123.test.%s.", strings.TrimSuffix(envZone, ".")), Data: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				libdns.RR{Type: "TXT", Name: "123.test", Data: "test", TTL: ttl},
			},
		},
	}

	for _, c := range testCases {
		func() {
			result, err := p.AppendRecords(context.TODO(), envZone+".", c.records)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupRecords(t, p, result)

			if len(result) != len(c.records) {
				t.Fatalf("len(resilt) != len(c.records) => %d != %d", len(c.records), len(result))
			}

			for k, r := range result {
				rr := r.RR()
				exp := c.expected[k].RR()
				if rr.Type != exp.Type {
					t.Fatalf("r.Type != c.exptected[%d].Type => %s != %s", k, rr.Type, exp.Type)
				}
				if rr.Name != exp.Name {
					t.Fatalf("r.Name != c.exptected[%d].Name => %s != %s", k, rr.Name, exp.Name)
				}
				if rr.Data != exp.Data {
					t.Fatalf("r.Value != c.exptected[%d].Value => %s != %s", k, rr.Data, exp.Data)
				}
				if rr.TTL != exp.TTL {
					t.Fatalf("r.TTL != c.exptected[%d].TTL => %s != %s", k, rr.TTL, exp.TTL)
				}
			}
		}()
	}
}

func Test_DeleteRecords(t *testing.T) {
	p := &domainnameshop.Provider{
		APIToken:  envToken,
		APISecret: envSecret,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZone)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.RR().Name == record.RR().Name {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.RR().Name)
		}
	}
}

func Test_GetRecords(t *testing.T) {
	p := &domainnameshop.Provider{
		APIToken:  envToken,
		APISecret: envSecret,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZone)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.RR().Name == record.RR().Name {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.RR().Name)
		}
	}
}

// TODO: This one don't work right just yet
func Test_SetRecords(t *testing.T) {
	p := &domainnameshop.Provider{
		APIToken:  envToken,
		APISecret: envSecret,
	}

	existingRecords, _ := setupTestRecords(t, p)
	newTestRecords := []libdns.Record{
		libdns.RR{
			Type: "TXT",
			Name: "new_test1",
			Data: "new_test1",
			TTL:  ttl,
		},
		libdns.RR{
			Type: "TXT",
			Name: "new_test2",
			Data: "new_test2",
			TTL:  ttl,
		},
	}

	allRecords := append(existingRecords, newTestRecords...)

	var test1 = allRecords[0].RR()
	test1.Data = "new_value"
	allRecords[0] = test1

	records, err := p.SetRecords(context.TODO(), envZone, allRecords)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupRecords(t, p, records)

	if len(records) != len(allRecords) {
		t.Fatalf("len(records) != len(allRecords) => %d != %d", len(records), len(allRecords))
	}
	var test2 = records[0].RR()

	if test2.Data != "new_value" {
		t.Fatalf(`records[0].Value != "new_value" => %s != "new_value"`, test2.Data)
	}
}
