package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// xmlErrorResponse returns a standard IAM XML error response body.
func xmlErrorResponse(code, message string) string {
	return `<ErrorResponse><Error><Code>` + code + `</Code><Message>` + message + `</Message></Error></ErrorResponse>`
}

// --- CreateUser ---

func TestCreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateUser" {
			t.Errorf("expected Action=CreateUser, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>testuser</UserName><UserId>AIDTEST123</UserId><Arn>arn:aws:iam::123:user/testuser</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	user, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.UserName != "testuser" {
		t.Errorf("expected UserName=testuser, got %s", user.UserName)
	}
	if user.UserId != "AIDTEST123" {
		t.Errorf("expected UserId=AIDTEST123, got %s", user.UserId)
	}
	if user.Arn != "arn:aws:iam::123:user/testuser" {
		t.Errorf("expected Arn=arn:aws:iam::123:user/testuser, got %s", user.Arn)
	}
	if user.Path != "/" {
		t.Errorf("expected Path=/, got %s", user.Path)
	}
}

// --- GetUser ---

func TestGetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GetUser" {
			t.Errorf("expected Action=GetUser, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetUserResponse><GetUserResult><User><UserName>testuser</UserName><UserId>AIDTEST123</UserId><Arn>arn:aws:iam::123:user/testuser</Arn><Path>/</Path></User></GetUserResult></GetUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	user, err := client.GetUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user == nil {
		t.Fatal("GetUser returned nil user")
	}
	if user.UserName != "testuser" {
		t.Errorf("expected UserName=testuser, got %s", user.UserName)
	}
	if user.UserId != "AIDTEST123" {
		t.Errorf("expected UserId=AIDTEST123, got %s", user.UserId)
	}
	if user.Arn != "arn:aws:iam::123:user/testuser" {
		t.Errorf("expected Arn=arn:aws:iam::123:user/testuser, got %s", user.Arn)
	}
}

func TestGetUserNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The user with name testuser cannot be found.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	user, err := client.GetUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("GetUser with NoSuchEntity should return nil error, got: %v", err)
	}
	if user != nil {
		t.Errorf("GetUser with NoSuchEntity should return nil user, got: %+v", user)
	}
}

// --- DeleteUser ---

func TestDeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteUser" {
			t.Errorf("expected Action=DeleteUser, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}
}

// --- PutUserPolicy ---

func TestPutUserPolicy(t *testing.T) {
	policyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "PutUserPolicy" {
			t.Errorf("expected Action=PutUserPolicy, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		if got := r.Form.Get("PolicyName"); got != "mypolicy" {
			t.Errorf("expected PolicyName=mypolicy, got %s", got)
		}
		if got := r.Form.Get("PolicyDocument"); got != policyDoc {
			t.Errorf("expected PolicyDocument=%s, got %s", policyDoc, got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.PutUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy", policyDoc)
	if err != nil {
		t.Fatalf("PutUserPolicy returned error: %v", err)
	}
}

// --- GetUserPolicy ---

func TestGetUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GetUserPolicy" {
			t.Errorf("expected Action=GetUserPolicy, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		if got := r.Form.Get("PolicyName"); got != "mypolicy" {
			t.Errorf("expected PolicyName=mypolicy, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetUserPolicyResponse><GetUserPolicyResult><UserName>testuser</UserName><PolicyName>mypolicy</PolicyName><PolicyDocument>%7B%22Version%22%3A%222012-10-17%22%7D</PolicyDocument></GetUserPolicyResult></GetUserPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	doc, err := client.GetUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("GetUserPolicy returned error: %v", err)
	}

	expected := `{"Version":"2012-10-17"}`
	if doc != expected {
		t.Errorf("expected URL-decoded policy document %q, got %q", expected, doc)
	}
}

func TestGetUserPolicyNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The user policy with name mypolicy cannot be found.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	doc, err := client.GetUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("GetUserPolicy with NoSuchEntity should return nil error, got: %v", err)
	}
	if doc != "" {
		t.Errorf("GetUserPolicy with NoSuchEntity should return empty string, got: %q", doc)
	}
}

// --- DeleteUserPolicy ---

func TestDeleteUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteUserPolicy" {
			t.Errorf("expected Action=DeleteUserPolicy, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		if got := r.Form.Get("PolicyName"); got != "mypolicy" {
			t.Errorf("expected PolicyName=mypolicy, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("DeleteUserPolicy returned error: %v", err)
	}
}

// --- CreateUserAccessKey ---

func TestCreateUserAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateAccessKey" {
			t.Errorf("expected Action=CreateAccessKey, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateAccessKeyResponse><CreateAccessKeyResult><AccessKey><UserName>testuser</UserName><AccessKeyId>AKIATEST</AccessKeyId><SecretAccessKey>secret123</SecretAccessKey><Status>Active</Status></AccessKey></CreateAccessKeyResult></CreateAccessKeyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	key, err := client.CreateUserAccessKey(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("CreateUserAccessKey returned error: %v", err)
	}
	if key.UserName != "testuser" {
		t.Errorf("expected UserName=testuser, got %s", key.UserName)
	}
	if key.AccessKeyId != "AKIATEST" {
		t.Errorf("expected AccessKeyId=AKIATEST, got %s", key.AccessKeyId)
	}
	if key.SecretAccessKey != "secret123" {
		t.Errorf("expected SecretAccessKey=secret123, got %s", key.SecretAccessKey)
	}
	if key.Status != "Active" {
		t.Errorf("expected Status=Active, got %s", key.Status)
	}
}

// --- ListUserAccessKeys ---

func TestListUserAccessKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "ListAccessKeys" {
			t.Errorf("expected Action=ListAccessKeys, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<ListAccessKeysResponse><ListAccessKeysResult><AccessKeyMetadata><member><UserName>testuser</UserName><AccessKeyId>AKIATEST1</AccessKeyId><Status>Active</Status></member><member><UserName>testuser</UserName><AccessKeyId>AKIATEST2</AccessKeyId><Status>Inactive</Status></member></AccessKeyMetadata></ListAccessKeysResult></ListAccessKeysResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	keys, err := client.ListUserAccessKeys(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("ListUserAccessKeys returned error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 access keys, got %d", len(keys))
	}
	if keys[0].AccessKeyId != "AKIATEST1" {
		t.Errorf("expected first key AccessKeyId=AKIATEST1, got %s", keys[0].AccessKeyId)
	}
	if keys[0].Status != "Active" {
		t.Errorf("expected first key Status=Active, got %s", keys[0].Status)
	}
	if keys[1].AccessKeyId != "AKIATEST2" {
		t.Errorf("expected second key AccessKeyId=AKIATEST2, got %s", keys[1].AccessKeyId)
	}
	if keys[1].Status != "Inactive" {
		t.Errorf("expected second key Status=Inactive, got %s", keys[1].Status)
	}
}

// --- DeleteUserAccessKey ---

func TestDeleteUserAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteAccessKey" {
			t.Errorf("expected Action=DeleteAccessKey, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		if got := r.Form.Get("AccessKeyId"); got != "AKIATEST" {
			t.Errorf("expected AccessKeyId=AKIATEST, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteUserAccessKey(context.Background(), "test-ak", "test-sk", "testuser", "AKIATEST")
	if err != nil {
		t.Fatalf("DeleteUserAccessKey returned error: %v", err)
	}
}

// --- CreateGroup ---

func TestCreateGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateGroup" {
			t.Errorf("expected Action=CreateGroup, got %s", got)
		}
		if got := r.Form.Get("GroupName"); got != "testgroup" {
			t.Errorf("expected GroupName=testgroup, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateGroupResponse><CreateGroupResult><Group><GroupName>testgroup</GroupName><GroupId>AGPTEST123</GroupId><Arn>arn:aws:iam::123:group/testgroup</Arn><Path>/</Path></Group></CreateGroupResult></CreateGroupResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	group, err := client.CreateGroup(context.Background(), "test-ak", "test-sk", "testgroup")
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	if group.GroupName != "testgroup" {
		t.Errorf("expected GroupName=testgroup, got %s", group.GroupName)
	}
	if group.GroupId != "AGPTEST123" {
		t.Errorf("expected GroupId=AGPTEST123, got %s", group.GroupId)
	}
	if group.Arn != "arn:aws:iam::123:group/testgroup" {
		t.Errorf("expected Arn=arn:aws:iam::123:group/testgroup, got %s", group.Arn)
	}
	if group.Path != "/" {
		t.Errorf("expected Path=/, got %s", group.Path)
	}
}

// --- GetGroup ---

func TestGetGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GetGroup" {
			t.Errorf("expected Action=GetGroup, got %s", got)
		}
		if got := r.Form.Get("GroupName"); got != "testgroup" {
			t.Errorf("expected GroupName=testgroup, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetGroupResponse><GetGroupResult><Group><GroupName>testgroup</GroupName><GroupId>AGPTEST123</GroupId><Arn>arn:aws:iam::123:group/testgroup</Arn><Path>/</Path></Group><Users><member><UserName>user1</UserName><UserId>AID001</UserId><Arn>arn:aws:iam::123:user/user1</Arn><Path>/</Path></member><member><UserName>user2</UserName><UserId>AID002</UserId><Arn>arn:aws:iam::123:user/user2</Arn><Path>/</Path></member></Users></GetGroupResult></GetGroupResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	group, users, err := client.GetGroup(context.Background(), "test-ak", "test-sk", "testgroup")
	if err != nil {
		t.Fatalf("GetGroup returned error: %v", err)
	}
	if group == nil {
		t.Fatal("GetGroup returned nil group")
	}
	if group.GroupName != "testgroup" {
		t.Errorf("expected GroupName=testgroup, got %s", group.GroupName)
	}
	if group.GroupId != "AGPTEST123" {
		t.Errorf("expected GroupId=AGPTEST123, got %s", group.GroupId)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].UserName != "user1" {
		t.Errorf("expected first user UserName=user1, got %s", users[0].UserName)
	}
	if users[0].UserId != "AID001" {
		t.Errorf("expected first user UserId=AID001, got %s", users[0].UserId)
	}
	if users[1].UserName != "user2" {
		t.Errorf("expected second user UserName=user2, got %s", users[1].UserName)
	}
}

func TestGetGroupNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The group with name testgroup cannot be found.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	group, users, err := client.GetGroup(context.Background(), "test-ak", "test-sk", "testgroup")
	if err != nil {
		t.Fatalf("GetGroup with NoSuchEntity should return nil error, got: %v", err)
	}
	if group != nil {
		t.Errorf("GetGroup with NoSuchEntity should return nil group, got: %+v", group)
	}
	if users != nil {
		t.Errorf("GetGroup with NoSuchEntity should return nil users, got: %+v", users)
	}
}

func TestGetGroupEmptyUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetGroupResponse><GetGroupResult><Group><GroupName>emptygroup</GroupName><GroupId>AGPEMPTY</GroupId><Arn>arn:aws:iam::123:group/emptygroup</Arn><Path>/</Path></Group><Users></Users></GetGroupResult></GetGroupResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	group, users, err := client.GetGroup(context.Background(), "test-ak", "test-sk", "emptygroup")
	if err != nil {
		t.Fatalf("GetGroup returned error: %v", err)
	}
	if group == nil {
		t.Fatal("GetGroup returned nil group")
	}
	if group.GroupName != "emptygroup" {
		t.Errorf("expected GroupName=emptygroup, got %s", group.GroupName)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestGetGroupPaginated(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		calls++
		marker := r.Form.Get("Marker")
		w.WriteHeader(200)
		switch marker {
		case "":
			// First page: truncated, one member, hands out a marker.
			_, _ = w.Write([]byte(`<GetGroupResponse><GetGroupResult><Group><GroupName>big</GroupName><GroupId>AGP1</GroupId><Arn>arn:aws:iam::123:group/big</Arn><Path>/</Path></Group><Users><member><UserName>user1</UserName><UserId>AID001</UserId><Arn>arn:aws:iam::123:user/user1</Arn><Path>/</Path></member></Users><IsTruncated>true</IsTruncated><Marker>page2</Marker></GetGroupResult></GetGroupResponse>`))
		case "page2":
			// Final page: not truncated, second member.
			_, _ = w.Write([]byte(`<GetGroupResponse><GetGroupResult><Group><GroupName>big</GroupName><GroupId>AGP1</GroupId><Arn>arn:aws:iam::123:group/big</Arn><Path>/</Path></Group><Users><member><UserName>user2</UserName><UserId>AID002</UserId><Arn>arn:aws:iam::123:user/user2</Arn><Path>/</Path></member></Users><IsTruncated>false</IsTruncated></GetGroupResult></GetGroupResponse>`))
		default:
			t.Errorf("unexpected Marker=%s", marker)
		}
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	group, users, err := client.GetGroup(context.Background(), "test-ak", "test-sk", "big")
	if err != nil {
		t.Fatalf("GetGroup returned error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 paginated requests, got %d", calls)
	}
	if group == nil || group.GroupName != "big" {
		t.Fatalf("expected group big, got %+v", group)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users across pages, got %d", len(users))
	}
	if users[0].UserName != "user1" || users[1].UserName != "user2" {
		t.Errorf("expected [user1 user2], got [%s %s]", users[0].UserName, users[1].UserName)
	}
}

// --- DeleteGroup ---

func TestDeleteGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteGroup" {
			t.Errorf("expected Action=DeleteGroup, got %s", got)
		}
		if got := r.Form.Get("GroupName"); got != "testgroup" {
			t.Errorf("expected GroupName=testgroup, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteGroup(context.Background(), "test-ak", "test-sk", "testgroup")
	if err != nil {
		t.Fatalf("DeleteGroup returned error: %v", err)
	}
}

// --- AddUserToGroup ---

func TestAddUserToGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "AddUserToGroup" {
			t.Errorf("expected Action=AddUserToGroup, got %s", got)
		}
		if got := r.Form.Get("GroupName"); got != "testgroup" {
			t.Errorf("expected GroupName=testgroup, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.AddUserToGroup(context.Background(), "test-ak", "test-sk", "testgroup", "testuser")
	if err != nil {
		t.Fatalf("AddUserToGroup returned error: %v", err)
	}
}

// --- RemoveUserFromGroup ---

func TestRemoveUserFromGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Action"); got != "RemoveUserFromGroup" {
			t.Errorf("expected Action=RemoveUserFromGroup, got %s", got)
		}
		if got := r.Form.Get("GroupName"); got != "testgroup" {
			t.Errorf("expected GroupName=testgroup, got %s", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("expected UserName=testuser, got %s", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.RemoveUserFromGroup(context.Background(), "test-ak", "test-sk", "testgroup", "testuser")
	if err != nil {
		t.Fatalf("RemoveUserFromGroup returned error: %v", err)
	}
}

// --- Error Handling ---

func TestServerErrorReturnsXMLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "Something went wrong")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "InternalFailure") {
		t.Errorf("expected error to contain 'InternalFailure', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("expected error to contain 'Something went wrong', got: %s", err.Error())
	}
}

func TestServerErrorNonXMLBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte("Service Unavailable"))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err == nil {
		t.Fatal("expected error from 503 response, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected error to contain status code '503', got: %s", err.Error())
	}
}

func TestDeleteUserServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "User is not authorized")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err == nil {
		t.Fatal("expected error from 403 response, got nil")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected error to contain 'AccessDenied', got: %s", err.Error())
	}
}

// --- Request Method and Content-Type Verification ---

func TestRequestIsPostWithFormContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type=application/x-www-form-urlencoded, got %s", contentType)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>testuser</UserName><UserId>AID1</UserId><Arn>arn:aws:iam::1:user/testuser</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionParameterIncluded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.Form.Get("Version"); got != "2010-05-08" {
			t.Errorf("expected Version=2010-05-08, got %s", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>x</UserName><UserId>X</UserId><Arn>arn</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
