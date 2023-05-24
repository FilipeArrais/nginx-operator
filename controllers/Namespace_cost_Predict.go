package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func PredictCostRequest() string {

	url := "http://kubecost-cost-analyzer.prometheus:9090/model/prediction/speccost?clusterID=cluster-one&defaultNamespace=teste"

	yamlApp, err2 := content.ReadFile("app/testspecs.yaml")
	if err2 != nil {
		panic(err2)
	}

	reqCost, err2 := http.NewRequest("POST", url, bytes.NewBuffer(yamlApp))
	if err2 != nil {
		panic(err2)
	}

	reqCost.Header.Set("Content-Type", "application/yaml")

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err2 := client.Do(reqCost)
	if err2 != nil {
		panic(err2)
	}

	defer resp.Body.Close()

	// Read the response body
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		panic(err2)
	}

	// Print the response body
	fmt.Println(string(body))

	return string(body)

}

// Returns total month Rate cost after the deployment of the yaml files passed
func decodeJsonPreditApiKubecost(jsonStr string) float64 {

	var data []map[string]interface{}

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		fmt.Println("Could not Unmarshal")
	}

	var total float64

	for _, item := range data {
		costAfter := item["costAfter"].(map[string]interface{})
		totalMonthlyRate := costAfter["totalMonthlyRate"].(float64)
		total = total + totalMonthlyRate
		fmt.Println(totalMonthlyRate)
	}

	return total

}
