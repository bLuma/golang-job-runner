package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func marshalJSONAsString(v interface{}) string {
	data, err := json.Marshal(v) // , "", "  ") /* Indent */
	if err != nil {
		log.Fatalln(err)
		return ""
	}

	return string(data)
}

func marshalJSONAndWrite(w io.Writer, v interface{}) bool {
	if httpw, ok := w.(http.ResponseWriter); ok {
		httpw.Header().Set("Content-Type", "application/json")
	}

	data, err := json.Marshal(v) // , "", "  ") /* Indent */
	if err != nil {
		log.Fatalln(err)
		return false
	}

	_, err = w.Write(data)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	return true
}

func unmarshalJSON(data []byte, v interface{}) bool {
	err := json.Unmarshal(data, v)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	return true
}
