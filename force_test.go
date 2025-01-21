package simpleforce

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

var (
	sfUser           = os.ExpandEnv("${SF_USER}")
	sfPass           = os.ExpandEnv("${SF_PASS}")
	sfToken          = os.ExpandEnv("${SF_TOKEN}")
	sfTrustIp        = os.ExpandEnv("${SF_TRUST_IP}")
	sfCustomEndPoint = os.ExpandEnv("${SF_CUSTOM_ENDPOINT}")
	sfURL            = func() string {
		if os.ExpandEnv("${SF_URL}") != "" {
			return os.ExpandEnv("${SF_URL}")
		} else {
			return DefaultURL
		}
	}()
)

func checkCredentialsAndSkip(t *testing.T) {
	if sfUser == "" || sfPass == "" {
		log.Println(logPrefix, "SF_USER, SF_PASS environment variables are not set.")
		t.Skip()
	}
}

func requireClient(ctx context.Context, t *testing.T, skippable bool) *Client {
	if skippable {
		checkCredentialsAndSkip(t)
	}

	client, err := NewClient(ctx, sfURL, DefaultClientID, DefaultAPIVersion)
	if err != nil {
		t.Error(err)
		return nil
	}

	if client == nil {
		t.Fatal()
		return nil
	}
	err = client.LoginPassword(ctx, sfUser, sfPass, sfToken)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestClient_LoginPassword(t *testing.T) {
	ctx := context.Background()

	checkCredentialsAndSkip(t)

	client, err := NewClient(ctx, sfURL, DefaultClientID, DefaultAPIVersion)
	if err != nil {
		t.Fatal(err)
	}

	if client == nil {
		t.Fatal()
	}

	// Use token
	err = client.LoginPassword(ctx, sfUser, sfPass, sfToken)
	if err != nil {
		t.Fatal(err)
	} else {
		log.Println(logPrefix, "sessionID:", client.sessionID)
	}

	err = client.LoginPassword(ctx, "__INVALID_USER__", "__INVALID_PASS__", "__INVALID_TOKEN__")
	if err == nil {
		t.Fatal(err)
	}
}

func TestClient_LoginPasswordNoToken(t *testing.T) {
	ctx := context.Background()

	if sfTrustIp == "" {
		log.Println(logPrefix, "SF_TRUST_IP environment variable is not set.")
		t.Skip()
	}

	checkCredentialsAndSkip(t)

	client, err := NewClient(ctx, sfURL, DefaultClientID, DefaultAPIVersion)
	if err != nil {
		t.Error(err)
	}

	if client == nil {
		t.Fatal()
	}

	// Trusted IP must be configured AND the request must be initiated from the trusted IP range.
	err = client.LoginPassword(ctx, sfUser, sfPass, "")
	if err != nil {
		t.Fatal(err)
	} else {
		log.Println(logPrefix, "sessionID:", client.sessionID)
	}
}

func TestClient_LoginOAuth(t *testing.T) {

}

func TestClient_Query(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	q := "SELECT Id,LastModifiedById,LastModifiedDate,ParentId,CommentBody FROM CaseComment"
	result, err := client.Query(ctx, q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.FailNow()
	}

	log.Println(logPrefix, result.TotalSize, result.Done, result.NextRecordsURL)
	if result.TotalSize < 1 {
		log.Println(logPrefix, "no records returned.")
		t.FailNow()
	}
	for _, record := range result.Records {
		if record.Type() != "CaseComment" {
			t.Fatal("invalid record type")
		}
	}
}

func TestClient_Query2(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	q := "Select id,createdbyid,parentid,parent.casenumber,parent.subject,createdby.name,createdby.alias FROM casecomment"
	result, err := client.Query(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) > 0 {
		comment1 := &result.Records[0]
		case1, err := comment1.SObjectField(ctx, "Case", "Parent").Get(ctx)

		if err != nil {
			t.Fatal(err)
		}

		if comment1.StringField("ParentId") != case1.ID() {
			t.Fatal("invalid parent id")
		}
	}
}

func TestClient_Query3(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	q := "SELECT Id FROM CaseComment WHERE CommentBody = 'This comment is created by simpleforce & used for testing'"
	result, err := client.Query(ctx, q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.FailNow()
	}

	log.Println(logPrefix, result.TotalSize, result.Done, result.NextRecordsURL)
	if result.TotalSize < 1 {
		log.Println(logPrefix, "no records returned.")
		t.FailNow()
	}
	for _, record := range result.Records {
		if record.Type() != "CaseComment" {
			t.Fatal("invalid record type")
		}
	}
}

func TestClient_ApexREST(t *testing.T) {
	ctx := context.Background()

	if sfCustomEndPoint == "" {
		t.Skip("SF_CUSTOM_ENDPOINT environment variable is not set.")
		return
	}

	client := requireClient(ctx, t, true)

	endpoint := "services/apexrest/my-custom-endpoint"
	result, err := client.ApexREST(ctx, endpoint, http.MethodPost, strings.NewReader(`{"my-property": "my-value"}`))
	if err != nil {
		log.Println(logPrefix, "request failed,", err)
		t.Fatal(err)
	}

	log.Println(logPrefix, string(result))
}

func TestClient_QueryLike(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	q := "Select Id, createdby.name, subject from case where subject like '%simpleforce%'"
	result, err := client.Query(ctx, q)
	if err != nil {
		t.FailNow()
	}
	if len(result.Records) > 0 {
		case0 := &result.Records[0]
		if !strings.Contains(case0.StringField("Subject"), "simpleforce") {
			t.FailNow()
		}
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
