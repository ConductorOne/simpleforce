package simpleforce

import (
	"context"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSObject_AttributesField(t *testing.T) {
	obj := &SObject{}
	if obj.AttributesField() != nil {
		t.Fail()
	}

	obj.setType("Case")
	if obj.AttributesField().Type != "Case" {
		t.Fail()
	}

	obj.setType("")
	if obj.AttributesField().Type != "" {
		t.Fail()
	}
}

func TestSObject_Type(t *testing.T) {
	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "Case"},
	}
	if obj.Type() != "Case" {
		t.Fail()
	}

	obj.setType("CaseComment")
	if obj.Type() != "CaseComment" {
		t.Fail()
	}
}

func TestSObject_InterfaceField(t *testing.T) {
	obj := &SObject{}
	if obj.InterfaceField("test_key") != nil {
		t.Fail()
	}

	(*obj)["test_key"] = "hello"
	if obj.InterfaceField("test_key") == nil {
		t.Fail()
	}
}

func TestSObject_SObjectField(t *testing.T) {
	ctx := context.Background()

	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "CaseComment"},
		"ParentId":           "__PARENT_ID__",
	}

	// Positive checks
	caseObj := obj.SObjectField(ctx, "Case", "ParentId")
	if caseObj.Type() != "Case" {
		log.Println("Type mismatch")
		t.Fail()
	}
	if caseObj.StringField("Id") != "__PARENT_ID__" {
		log.Println("ID mismatch")
		t.Fail()
	}

	// Negative checks
	userObj := obj.SObjectField(ctx, "User", "OwnerId")
	if userObj != nil {
		log.Println("Nil mismatch")
		t.Fail()
	}
}

func TestSObject_Describe(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)
	meta := client.SObject("Case").Describe(ctx)
	if meta == nil {
		t.FailNow()
	} else {
		if (*meta)["name"].(string) != "Case" {
			t.Fail()
		}
	}
}

func TestSObject_Get(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	// Search for a valid Case ID first.
	queryResult, err := client.Query(ctx, "SELECT Id,OwnerId,Subject FROM CASE")
	if err != nil || queryResult == nil {
		t.Logf("Query failed: %v", err)
		t.FailNow()
	}
	if queryResult.TotalSize < 1 {
		t.FailNow()
	}
	oid := queryResult.Records[0].ID()
	ownerID := queryResult.Records[0].StringField("OwnerId")

	// Positive
	obj, err := client.SObject("Case").Get(ctx, oid)
	if err != nil {
		t.Fatal(err)
	}

	if obj.ID() != oid || obj.StringField("OwnerId") != ownerID {
		t.Fail()
	}

	// Positive 2
	obj = client.SObject("Case")
	if obj.StringField("OwnerId") != "" {
		t.Fail()
	}
	obj.setID(oid)
	_, _ = obj.Get(ctx)
	if obj.ID() != oid || obj.StringField("OwnerId") != ownerID {
		t.Fail()
	}

	// Negative 1
	obj, err = client.SObject("Case").Get(ctx, "non-exist-id")
	if err == nil {
		t.Fatal(err)
	}

	if obj != nil {
		t.Fail()
	}

	// Negative 2
	obj = &SObject{}

	newObject, _ := obj.Get(ctx)

	if newObject != nil {
		t.Fail()
	}
}

func TestSObject_Create(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	// Positive
	case1 := client.SObject("Case")
	case1Result, err := case1.Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("Comments", "This case is created by simpleforce").
		Create(ctx)

	if err != nil {
		t.Fatal(err)
	}

	if case1Result == nil || case1Result.ID() == "" || case1Result.Type() != case1.Type() {
		t.Fail()
	} else {

		objInner, err := case1Result.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}

		log.Println(logPrefix, "Case created,", objInner.StringField("CaseNumber"))
	}

	// Positive 2
	caseComment1 := client.SObject("CaseComment")
	caseComment1Result, err := caseComment1.Set("ParentId", case1Result.ID()).
		Set("CommentBody", "This comment is created by simpleforce & used for testing").
		Set("IsPublished", true).
		Create(ctx)

	if err != nil {
		t.Fatal(err)
	}

	objAssert, err := caseComment1Result.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if objAssert.SObjectField(ctx, "Case", "ParentId").ID() != case1Result.ID() {
		t.Fail()
	} else {
		log.Println(logPrefix, "CaseComment created,", caseComment1Result.ID())
	}

	// Negative: object without type.
	obj := client.SObject()
	create, err := obj.Create(ctx)
	if errors.Is(err, ErrNoTypeIdClientOrId) == false {
		t.Fatal("Expected ErrNoTypeIdClientOrId")
	}

	if create != nil {
		t.Fatal("Expected nil")
	}

	// Negative: object without client.
	obj = &SObject{}

	objAssert, err = obj.Create(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: Invalid type
	obj = client.SObject("__SOME_INVALID_TYPE__")
	objAssert, err = obj.Create(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: Invalid field
	obj = client.SObject("Case").Set("__SOME_INVALID_FIELD__", "")

	objAssert, err = obj.Create(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}
}

func TestSObject_Update(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	obj, err := client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Create(ctx)

	if err != nil {
		t.Fatal(err)
	}

	objUpdated, err := obj.
		Set("Subject", "Case subject updated by simpleforce").
		Update(ctx)

	if err != nil {
		t.Fatal(err)
	}

	objAssert, err := objUpdated.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Positive
	if objAssert.StringField("Subject") != "Case subject updated by simpleforce" {
		t.Fail()
	}
}

func TestSObject_Upsert(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	// Positive create new object through upsert
	case1 := client.SObject("Case")
	case1Result, err := case1.Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("Comments", "This case is created by simpleforce").
		Set("ExternalIDField", "customExtIdField__c").
		Set("customExtIdField__c", uuid.NewString()).
		Upsert(ctx)

	if err != nil {
		t.Fatal(err)
	}

	if case1Result == nil || case1Result.ID() == "" || case1Result.Type() != case1.Type() {
		t.Fail()
	} else {
		objAssert, err := case1Result.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}

		log.Println(logPrefix, "Case created,", objAssert.StringField("CaseNumber"))
	}

	// Positive update existing object through upsert
	case2 := client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("customExtIdField__c", uuid.NewString())
	case2Result, err := case2.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = case2.
		Set("Subject", "Case subject updated by simpleforce").
		Set("ExternalIDField", "customExtIdField__c").
		Upsert(ctx)
	if err != nil {
		t.Fatal(err)
	}

	objAssert, err := case2Result.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if objAssert.StringField("Subject") != "Case subject updated by simpleforce" {
		t.Fail()
	} else {
		log.Println(logPrefix, "Case updated,", objAssert.StringField("CaseNumber"))
	}

	// Negative: object without type.
	obj := client.SObject()

	objAssert, err = obj.Upsert(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: object without client.
	obj = &SObject{}
	objAssert, err = obj.Upsert(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: Invalid type
	obj = client.SObject("__SOME_INVALID_TYPE__").
		Set("ExternalIDField", "customExtIdField__c").
		Set("customExtIdField__c", uuid.NewString())

	objAssert, err = obj.Upsert(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: Invalid field
	obj = client.SObject("Case").
		Set("ExternalIDField", "customExtIdField__c").
		Set("customExtIdField__c", uuid.NewString()).
		Set("__SOME_INVALID_FIELD__", "")

	objAssert, err = obj.Upsert(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}

	// Negative: Missing ext ID
	obj = client.SObject("Case").
		Set("ExternalIDField", "customExtIdField__c")

	objAssert, err = obj.Upsert(ctx)
	if err == nil {
		t.Fatal("Expected error")
	}

	if objAssert != nil {
		t.Fatal("Expected nil")
	}
}

func TestSObject_Delete(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	// Positive: create a case first then delete it and verify if it is gone.
	obj, err := client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Create(ctx)

	if err != nil {
		t.Fatal(err)
	}

	case1, err := obj.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if case1 == nil || case1.ID() == "" {
		t.Fatal("Invalid case")
	}

	caseID := case1.ID()
	if case1.Delete(ctx) != nil {
		t.Fatal("Failed to delete case")
	}

	if case1.Delete(ctx) == nil {
		t.Fatal("Expected error, should not delete twice")
	}

	err = uhttp.ClearCaches(ctx)
	if err != nil {
		t.Fatal(err)
	}

	case1, err = client.SObject("Case").Get(ctx, caseID)

	if status.Code(err) != codes.NotFound {
		t.Fatal("Case still exists")
	}

	if case1 != nil {
		t.Fatal("Case still exists")
	}
}

// TestSObject_GetUpdate validates updating of existing records.
func TestSObject_GetUpdate(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	// Create a new case first.
	obj1, err := client.SObject("Case").
		Set("Subject", "Original").
		Create(ctx)

	if err != nil {
		t.Fatal(err)
	}

	case1, err := obj1.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Query the case by ID, then update the Subject.

	obj2, err := client.SObject("Case").
		Get(ctx, case1.ID())

	if err != nil {
		t.Fatal(err)
	}

	obj2Copy := &SObject{
		sobjectClientKey:              obj2.client(),
		sobjectIDKey:                  obj2.ID(),
		sobjectExternalIDFieldNameKey: obj2.ExternalIDFieldName(),
		obj2.ExternalIDFieldName():    obj2.ExternalID(),
	}
	obj2Copy.setType(obj2.Type())

	case2, err := obj2Copy.
		Set("Subject", "Updated").
		Update(ctx)

	if err != nil {
		t.Fatal(err)
	}

	_, err = case2.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = uhttp.ClearCaches(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Query the case by ID again and check if the Subject has been updated.
	case3, err := client.
		SObject("Case").
		Get(ctx, case2.ID())

	if err != nil {
		t.Fatal(err)
	}

	if case3.StringField("Subject") != "Updated" {
		t.Fatal("Subject not updated")
	}
}
