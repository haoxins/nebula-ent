package ent

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	nebula "github.com/vesoft-inc/nebula-go/v3"
)

func TestEntityBuilder(t *testing.T) {
	spaceName := "test_space_entity_builder"
	err := prepareSpace(spaceName)
	if err != nil {
		t.Fatal(err)
	}

	hostAddr := nebula.HostAddress{Host: address, Port: port}
	conf, err := nebula.NewSessionPoolConf(
		username,
		password,
		[]nebula.HostAddress{hostAddr},
		spaceName,
	)
	if err != nil {
		t.Errorf("failed to create session pool config, %s", err.Error())
	}

	// create session pool
	sessionPool, err := nebula.NewSessionPool(*conf, nebula.DefaultLogger{})
	if err != nil {
		t.Fatal(err)
	}
	defer sessionPool.Close()

	schemaManager := nebula.NewSchemaManager(sessionPool)

	tagSchema := nebula.LabelSchema{
		Name: "user",
		Fields: []nebula.LabelFieldSchema{
			{
				Field:    "name",
				Type:     "string",
				Nullable: false,
			},
			{
				Field:    "age",
				Type:     "int",
				Nullable: true,
			},
		},
	}

	_, err = schemaManager.ApplyTag(tagSchema)
	if err != nil {
		t.Fatal(err)
	}

	edgeSchema := nebula.LabelSchema{
		Name: "friend",
		Fields: []nebula.LabelFieldSchema{
			{
				Field:    "created_at",
				Type:     "int64",
				Nullable: true,
			},
		},
	}

	_, err = schemaManager.ApplyEdge(edgeSchema)
	if err != nil {
		t.Fatal(err)
	}

	// waiting for the schema to be propagated
	time.Sleep(5 * time.Second)

	b := NewEntityBuilder("user").SetProp("name", "Bob").SetProp("age", 18).UpsertVertex("test1")
	assert.Equal(t, `UPSERT VERTEX ON user "test1" SET name = "Bob", age = 18;`, b.String())

	_, err = b.Exec(sessionPool)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewEntityBuilder("user").SetProp("name", `Lily "Double quotation"`).SetProp("age", 20).UpsertVertex("test2").Exec(sessionPool)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	b = NewEntityBuilder("friend").SetProp("created_at", now).UpsertEdge("test1", "test2")
	assert.Equal(t, fmt.Sprintf(`UPSERT EDGE ON friend "test1" -> "test2" SET created_at = %d;`, now), b.String())

	_, err = b.Exec(sessionPool)
	if err != nil {
		t.Fatal(err)
	}

	rs, err := sessionPool.ExecuteAndCheck(`FETCH PROP ON friend "test1" -> "test2" YIELD edge AS e;`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(rs.GetRows()))
}
