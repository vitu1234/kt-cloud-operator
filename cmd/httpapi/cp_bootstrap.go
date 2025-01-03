package httpapi

import (
	"bytes"
	"errors"
	"net/http"
	"time"

	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
)

func CheckControlPlaneMachineReady(machine *v1beta1.KTMachine) error {
	// Define the API URL
	apiURL := "http://" + machine.Status.AssignedPublicIps[0].IP + ":8000"

	// Set up the HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", apiURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		logger1.Error("Error creating GET VM request:", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("X-Auth-Token", token) // Replace with actual token

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		logger1.Error("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	// fmt.Println("Response Status:", resp.Status)
	// _, err = io.ReadAll(resp.Body)
	// if err != nil {
	// 	logger1.Error("Error reading response body:", err)
	// 	return err
	// }

	// logger1.Info("-----------------------------------------")
	// logger1.Info("Response Body Networks:", string(body))
	// logger1.Info("********************************")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger1.Info("Api Server is ready!")

		return nil

	} else {
		logger1.Error("GET request failed with status:", resp.Status)
		return errors.New("GET control-plane status request failed with status: " + resp.Status)
	}
}
