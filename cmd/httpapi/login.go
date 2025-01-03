package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// Meta API for object metadata
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"dcnlab.ssu.ac.kr/kt-cloud-operator/internal/cloudapi"
)

type LoginResponse struct {
	SubjectToken string `json:"subjectToken,omitempty"`
	Token        Token  `json:"token,omitempty"`
	Date         string `json:"date,omitempty"`
}

type Token struct {
	ExpiresAt string `json:"expiresAt,omitempty"`
	IsDomain  bool   `json:"isDomain,omitempty"`
}

// Structs for login
type AuthRequest struct {
	Auth Auth `json:"auth"`
}

type Auth struct {
	Identity Identity `json:"identity"`
	Scope    Scope    `json:"scope"`
}

type Identity struct {
	Methods  []string `json:"methods"`
	Password Password `json:"password"`
}

type Password struct {
	User User `json:"user"`
}

type User struct {
	Domain   Domain `json:"domain"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Domain struct {
	ID string `json:"id"`
}

type Scope struct {
	Project Project `json:"project"`
}

type Project struct {
	Domain Domain `json:"domain"`
	Name   string `json:"name"`
}

// end structs for login

// var Config cloudapi.Config
// var logger1 *zap.SugaredLogger

// func ProcessEnvVariables() {
// 	err := envconfig.Process("", &Config)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	err, logger1 = logger(Config.LogLevel)
// 	if err != nil {
// 		panic(err.Error())
// 	}

// 	logger1.Info("Processed Env Variables...")
// }

var Config cloudapi.Config
var logger1 *zap.SugaredLogger

func init() {
	Config, logger1 = ProcessEnvVariables()
}

func KTCloudLogin() {
	// ProcessEnvVariables()

	// Create an instance of the struct with your data
	authRequest := AuthRequest{
		Auth: Auth{
			Identity: Identity{
				Methods: []string{Config.IdentityMethods},
				Password: Password{
					User: User{
						Domain:   Domain{ID: Config.IdentityPasswordUserDomainId},
						Name:     Config.IdentityPasswordUserName,
						Password: Config.IdentityPassword,
					},
				},
			},
			Scope: Scope{
				Project: Project{
					Domain: Domain{ID: Config.ScopeProjectDomainId},
					Name:   Config.ScopeProjectName,
				},
			},
		},
	}

	// Marshal the struct to JSON
	payload, err := json.Marshal(authRequest)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Define the endpoint URL
	apiURL := Config.ApiBaseURL + Config.Zone + "/identity/auth/tokens"

	// Set up HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		logger1.Fatal("Error creating KT Cloud Auth API request:", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Fatal("Error sending KT Cloud Auth POST request:", err)
		return
	}
	defer resp.Body.Close()

	// Handle the response
	logger1.Info("Response Status:", resp.Status)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("POST request successful!")
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger1.Fatal("Error reading response body:", err)
			return
		}

		token := ""
		for key, values := range resp.Header {
			for _, value := range values {
				// fmt.Printf("%s: %s\n", key, value)
				if key == "X-Subject-Token" {
					logger1.Info("TOKEN: ", value)
					token = value
				}
			}
		}
		// Print the actual response body
		logger1.Info("Response Body:")
		logger1.Info(string(body))

		// create token object
		clientConfig, err := getRestConfig(Config.Kubeconfig)
		if err != nil {
			logger1.Errorf("Cannot prepare k8s client config: %v. Kubeconfig was: %s", err, Config.Kubeconfig)
			panic(err.Error())
		}
		// Set up a scheme (use runtime.Scheme from apimachinery)
		scheme := runtime.NewScheme()
		// Create Kubernetes client
		k8sClient, err := getClient(clientConfig, scheme)
		if err != nil {
			logger1.Fatalf("Failed to create Kubernetes client: %v", err)
		}

		// Use the client (example)
		logger1.Info("Kubernetes client successfully created:", k8sClient)
		createTokenObject(k8sClient, token, string(body))

	} else {
		logger1.Fatal("POST request to KT Cloud Auth failed with status: %s\n", resp.Status)

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger1.Fatal("Error reading response body:", err)
			return
		}

		// Print the actual response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	}
}

func createTokenObject(k8sClient client.Client, subjectToken, responseBody string) {

	// Define a map to hold the parsed data
	var parsedData map[string]interface{}

	// Parse the JSON
	err := json.Unmarshal([]byte(responseBody), &parsedData)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Navigate through the JSON to extract items
	token := parsedData["token"].(map[string]interface{})
	expiresAt := token["expires_at"]
	isDomain := token["is_domain"]
	// methods := token["methods"].([]interface{})
	// auditIDs := token["audit_ids"].([]interface{})
	// catalog := token["catalog"].([]interface{})
	// roles := token["roles"].([]interface{})
	// project := token["project"].(map[string]interface{})
	// user := token["user"].(map[string]interface{})

	// // Print extracted items
	// fmt.Println("Expires At:", expiresAt)
	// fmt.Println("Methods:", methods)
	// fmt.Println("Audit IDs:", auditIDs)
	// fmt.Println("Catalog Services:")
	// for _, service := range catalog {
	// 	serviceMap := service.(map[string]interface{})
	// 	fmt.Printf("  - Name: %s, Type: %s, Endpoints: %v\n", serviceMap["name"], serviceMap["type"], serviceMap["endpoints"])
	// }
	// fmt.Println("Roles:")
	// for _, role := range roles {
	// 	roleMap := role.(map[string]interface{})
	// 	fmt.Printf("  - Name: %s, ID: %s\n", roleMap["name"], roleMap["id"])
	// }
	// fmt.Println("Project Name:", project["name"])
	// fmt.Println("User Name:", user["name"])

	tokenObj := &v1beta1.KTSubjectToken{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ktcloudtoken",
			Namespace: "default",
		},
		Spec: v1beta1.KTSubjectTokenSpec{
			SubjectToken: subjectToken,
			Token: v1beta1.Token{
				ExpiresAt: expiresAt.(string),
				IsDomain:  isDomain.(bool),
			},
		},
		Status: v1beta1.KTSubjectTokenStatus{
			SubjectToken: subjectToken,
			Token: v1beta1.Token{
				ExpiresAt: expiresAt.(string),
				IsDomain:  isDomain.(bool),
			},
			CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000000Z"),
		},
	}
	ctx := context.Background()

	// Use the global k8sClient to create the custom resource
	// Check if the object already exists
	existingTokenObj := &v1beta1.KTSubjectToken{}
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      tokenObj.Name,
		Namespace: tokenObj.Namespace,
	}, existingTokenObj)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Object does not exist, create it
			logger1.Info("KTSubjectToken does not exist, creating a new one")
			err = k8sClient.Create(ctx, tokenObj)
			if err != nil {
				logger1.Errorf("Failed to create KTSubjectToken object: %v", err)
				return
			}
			logger1.Info("KTSubjectToken object created successfully!")
		} else {
			// Error fetching object
			logger1.Errorf("Failed to fetch KTSubjectToken object: %v", err)
			return
		}
	} else {
		// Object exists, update it
		logger1.Info("KTSubjectToken already exists, updating it")
		existingTokenObj.Status = tokenObj.Status
		err = k8sClient.Status().Update(ctx, existingTokenObj)
		if err != nil {
			logger1.Errorf("Failed to update KTSubjectToken object: %v", err)
			return
		}
		logger1.Info("KTSubjectToken object updated successfully!")
	}

}
