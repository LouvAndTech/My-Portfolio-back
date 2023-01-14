package main

import (
	"context"
	"fmt"
	"log"

	"github.com/machinebox/graphql"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	pbtypes "github.com/pocketbase/pocketbase/tools/types"
)

type GithubRepo struct {
	Name        string
	Url         string
	Owner       string
	AvatarUrl   string
	Description string
	UpdatedAt   string
	Stars       int
	Forks       int
	//Here place the languages request
}

func newRepo(name string, url string, owner string, avatarUrl string, description string, updatedAt string, stars int, forks int) GithubRepo {
	return GithubRepo{
		Name:        name,
		Url:         url,
		Owner:       owner,
		AvatarUrl:   avatarUrl,
		Description: description,
		UpdatedAt:   updatedAt,
		Stars:       stars,
		Forks:       forks,
	}
}

/** Update the github repo list on the database from the github api */
func UpdateGithub() {
	//Handle the error
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic occurred:", err)
		}
	}()
	//First fetch detail from github
	fmt.Println("Updating github")
	//New cleint for graphql
	graphqlClient := graphql.NewClient("https://api.github.com/graphql")
	//Request body
	graphqlRequest := graphql.NewRequest(`
	{
		viewer {
			login
			name
			#Repositories created by the viewer
			repositories(first:100,privacy:PUBLIC,ownerAffiliations:OWNER) {
				totalDiskUsage
				edges{
					node{
						name
						url
						owner {
							login
							avatarUrl
						}
						description
						updatedAt
						stargazerCount
            			forkCount
						#Here place the languages request
					}
				}
			}
		}
	}
    `)
	//Set the header
	graphqlRequest.Header.Set("Authorization", "bearer "+github_token)
	//Request header
	graphqlRequest.Header.Set("Content-Type", "application/json")
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)
	}
	//fmt.Println(graphqlResponse)

	//Parse the response and update the database
	for _, repo := range graphqlResponse.(map[string]interface{})["viewer"].(map[string]interface{})["repositories"].(map[string]interface{})["edges"].([]interface{}) {

		//for each repo in the response, create a new repo object

		//Handle empty descriptions assuming that every other field is not empty based on github requierements
		if (repo.(map[string]interface{})["node"].(map[string]interface{})["description"]) == nil {
			repo.(map[string]interface{})["node"].(map[string]interface{})["description"] = ""
		}
		//Create a new repo
		newRepo := newRepo(
			repo.(map[string]interface{})["node"].(map[string]interface{})["name"].(string),                                        //repos name
			repo.(map[string]interface{})["node"].(map[string]interface{})["url"].(string),                                         //repos url
			repo.(map[string]interface{})["node"].(map[string]interface{})["owner"].(map[string]interface{})["login"].(string),     //repos owner name
			repo.(map[string]interface{})["node"].(map[string]interface{})["owner"].(map[string]interface{})["avatarUrl"].(string), //repos owner avatar url (used as image)
			repo.(map[string]interface{})["node"].(map[string]interface{})["description"].(string),                                 //repos description
			repo.(map[string]interface{})["node"].(map[string]interface{})["updatedAt"].(string),                                   //repos last update date (used to sort)
			int(repo.(map[string]interface{})["node"].(map[string]interface{})["stargazerCount"].(float64)),                        //repos stars count
			int(repo.(map[string]interface{})["node"].(map[string]interface{})["forkCount"].(float64)),                             //repos forks count
		)
		//fmt.Println(newRepo)

		fmt.Println("Updating repo: " + newRepo.Name)

		//Get the collection

		defer func() {
			if err := recover(); err != nil {
				log.Println("panic occurred when trying to add repo to the db:", err)
			}
		}()

		//fetch collection
		collection, err := app.Dao().FindCollectionByNameOrId("github_projects")
		if err != nil {
			//if collection does not exist
			if collection == nil {
				fmt.Println("Collection does not exist")
				//Create a new collection
				crreateCollection()
				collection, err = app.Dao().FindCollectionByNameOrId("github_projects")
				if err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		}

		record, _ := app.Dao().FindRecordsByExpr(collection.Id, dbx.HashExp{"repo_name": newRepo.Name})
		if err != nil {
			panic(err)
		}

		var toAdd models.Record

		if len(record) == 0 { //the repo dose not exist in the database
			fmt.Println("The repo does not exist")
			//Create a new record
			toAdd = *models.NewRecord(collection)
		} else {
			toAdd = *record[0]
		}

		toAdd.Set("repo_name", newRepo.Name)
		toAdd.Set("link_to_repo", newRepo.Url)
		toAdd.Set("username", newRepo.Owner)
		toAdd.Set("image_link", newRepo.AvatarUrl)
		toAdd.Set("tags", "")
		toAdd.Set("description", newRepo.Description)
		toAdd.Set("stars", newRepo.Stars)
		toAdd.Set("fork", newRepo.Forks)
		toAdd.Set("contributor", "0")

		if err := app.Dao().SaveRecord(&toAdd); err != nil {
			panic(err)
		}

	}
}

func crreateCollection() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic occurred when trying to create the collection:", err)
		}
	}()

	collection := &models.Collection{}

	form := forms.NewCollectionUpsert(app, collection)
	form.Name = "github_projects"
	form.Type = models.CollectionTypeBase
	form.ListRule = pbtypes.Pointer("")
	form.ViewRule = pbtypes.Pointer("")
	form.CreateRule = nil
	form.UpdateRule = nil
	form.DeleteRule = nil
	form.Schema.AddField(&schema.SchemaField{
		Name:     "username",
		Type:     schema.FieldTypeText,
		Required: true,
		Unique:   false,
		Options:  &schema.TextOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "repo_name",
		Type:     schema.FieldTypeText,
		Required: true,
		Unique:   false,
		Options:  &schema.TextOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "image_link",
		Type:     schema.FieldTypeText,
		Required: false,
		Unique:   false,
		Options:  &schema.TextOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "tags",
		Type:     schema.FieldTypeJson,
		Required: false,
		Unique:   false,
		Options:  &schema.JsonOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "link_to_repo",
		Type:     schema.FieldTypeText,
		Required: true,
		Unique:   false,
		Options:  &schema.TextOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "description",
		Type:     schema.FieldTypeText,
		Required: true,
		Unique:   false,
		Options:  &schema.TextOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "stars",
		Type:     schema.FieldTypeNumber,
		Required: false,
		Unique:   false,
		Options:  &schema.NumberOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "fork",
		Type:     schema.FieldTypeNumber,
		Required: false,
		Unique:   false,
		Options:  &schema.NumberOptions{},
	})
	form.Schema.AddField(&schema.SchemaField{
		Name:     "contributor",
		Type:     schema.FieldTypeNumber,
		Required: false,
		Unique:   false,
		Options:  &schema.NumberOptions{},
	})

	// validate and submit (internally it calls app.Dao().SaveCollection(collection) in a transaction)
	if err := form.Submit(); err != nil {
		panic(err)
	}
}

//FOR LATER USE :

/*Language request
#Languages used in the repository
languages(first:100){
	totalCount
	totalSize
	edges{
		size
		node{
			name
			color
		}
	}
}
*/

/* Here is the next part that i'll be working on later :
#Repositories contributed to by the viewer
repositoriesContributedTo(
first: 100,
contributionTypes: [COMMIT, PULL_REQUEST]){
	totalCount,
	nodes {
		name
		owner {
			login
			avatarUrl
		}
		url,
		description,
		updatedAt
		languages(first: 100){
			totalCount,
			totalSize,
			edges{
				size,
				node {
					name
					color
				}
			}
		}
	}
	pageInfo {
		endCursor
		hasNextPage
	}
}
*/
