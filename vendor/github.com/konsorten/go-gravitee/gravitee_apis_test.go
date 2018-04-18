package gravitee

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestGetAllAPIs(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAllAPIs()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)
}

func TestGetAPIsByLabel(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	api, err := session.GetAPI(apis[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", api)

	meta, err := session.GetAPIMetadata(apis[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", meta)

	for _, m := range meta {
		if m.IsLocal() {
			md, err := session.GetLocalAPIMetadata(apis[0].ID, m.Key)
			if err != nil {
				t.Fatal(err)
			}

			fmt.Printf("%v\n", md)
		}
	}
}

func TestSetLocalAPIMetadata(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	key := fmt.Sprintf("unittest-%v", rand.Int63())
	val := fmt.Sprintf("vvvv#%v", rand.Int63())
	err = session.SetLocalAPIMetadata(apis[0].ID, key, val, ApiMetadataFormat_String)
	if err != nil {
		t.Fatal(err)
	}

	md, err := session.GetLocalAPIMetadata(apis[0].ID, key)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", md)

	if md.Value() != val {
		t.Fatal("Expected local value")
	}

	err = session.UnsetLocalAPIMetadata(apis[0].ID, key)
	if err != nil {
		t.Fatal(err)
	}

	notFound, err := session.GetLocalAPIMetadata(apis[0].ID, key)
	if err != nil {
		t.Fatal(err)
	}
	if notFound != nil {
		t.Fatal("Expected not found")
	}
}

func TestGetLocalAPIMetadataNotFound(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	notFound, err := session.GetLocalAPIMetadata(apis[0].ID, "d0esn0tex1st")
	if err != nil {
		t.Fatal(err)
	}
	if notFound != nil {
		t.Fatal("Expected not found")
	}
}

func TestUnsetLocalAPIMetadataNotFound(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	err = session.UnsetLocalAPIMetadata(apis[0].ID, "d0esn0tex1st")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeployAPI(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	err = session.DeployAPI(apis[0].ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddOrUpdateEndpoints(t *testing.T) {
	session := createTestSession(t)

	apis, err := session.GetAPIsByLabel("gravitee-go")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%v\n", apis)

	ep := []ApiDetailsEndpoint{
		MakeApiDetailsEndpoint("default", "http://klihgukjdvbfjgbjhdfb"),
		MakeApiDetailsEndpoint("default2", "http://skjfgjshdfshjdgfjhsgdhj"),
	}

	err = session.AddOrUpdateEndpoints(apis[0].ID, ep, true)
	if err != nil {
		t.Fatal(err)
	}
}
