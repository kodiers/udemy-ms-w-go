package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// TODO: call trip service
	defer r.Body.Close()
	if reqBody.UserId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	log.Println(reqBody)

	jsonBody, _ := json.Marshal(reqBody)
	reader := bytes.NewReader(jsonBody)

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tripService.Close()
	tripService.Client.PreviewTrip()
	resp, err := http.Post("http://trip-service:8083/preview", "application/json", reader)
	if err != nil {
		http.Error(w, "Cannot request trip-service", http.StatusBadRequest)
		return
	}

	defer resp.Body.Close()

	var responseBody any
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := contracts.APIResponse{
		Data: responseBody,
	}
	writeJson(w, http.StatusCreated, response)

}
