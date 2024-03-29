package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func AtualCostRequest(namespace string) string {

	url := "http://kubecost-cost-analyzer.prometheus:9090/model/allocation?window=30d&&filterNamespaces=" +
		namespace +
		"&&aggregate=namespace&&accumulate=true&&shareTenancyCosts=true"

	// Criação de uma requisição GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Erro ao criar a requisição:", err)
		panic(err)
	}

	// Criar uma instância do cliente HTTP
	client := &http.Client{}
	// Enviar a requisição
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Erro ao enviar a requisição:", err)
		panic(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Erro ao ler o corpo da resposta:", err)
		panic(err)
	}

	// Exibir a resposta
	//fmt.Println(string(body))
	return string(body)

}

// Returns total month Rate cost after the deployment of the yaml files passed
func decodeJsonAllocationApiKubecost(jsonStr string, namespace string) float64 {

	var data map[string]interface{}

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		fmt.Println("Could not Unmarshal")
	}

	// Extrair o totalCost do namespace

	namespaceDetails, ok := data["data"].([]interface{})[0].(map[string]interface{})[namespace].(map[string]interface{})
	if !ok {
		fmt.Println("Falha ao extrair namespace do JSON")
	}
	totalCost, ok := namespaceDetails["totalCost"].(float64)
	if !ok {
		fmt.Println("Falha ao extrair o campo 'totalCost' do JSON")
	}

	return totalCost

}
