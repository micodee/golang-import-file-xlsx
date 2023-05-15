package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/tealeg/xlsx"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/import", convertHandler).Methods("POST")

	port := "8000"
	fmt.Println("server running on port", port)
	http.ListenAndServe("localhost:"+port, router)
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	// Menerima file XLSX dari form-data
	err := r.ParseMultipartForm(10 << 20) // Maksimum ukuran file: 10MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Membaca file XLSX
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	xlFile, err := xlsx.OpenBinary(fileBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mengonversi file XLSX menjadi format CSV
	csvData := make([][]string, 0)
	for _, sheet := range xlFile.Sheets {
		for _, row := range sheet.Rows {
			csvRow := make([]string, 0)
			for _, cell := range row.Cells {
				csvRow = append(csvRow, cell.String())
			}
			csvData = append(csvData, csvRow)
		}
	}

	// Menulis data CSV ke file sementara
	tempFile, err := ioutil.TempFile("", "temp*.csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	csvWriter := csv.NewWriter(tempFile)
	err = csvWriter.WriteAll(csvData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	csvWriter.Flush()
	tempFile.Close()

	// Membaca data dari file CSV sementara
	tempFile, err = os.Open(tempFile.Name())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	csvReader := csv.NewReader(tempFile)
	jsonData, err := csvToJSON(csvReader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mengirimkan data JSON sebagai response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func csvToJSON(csvReader *csv.Reader) ([]byte, error) {
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	headers := records[0]
	jsonData := make([]map[string]string, 0)

	for _, row := range records[1:] {
		item := make(map[string]string)
		for i, value := range row {
			item[headers[i]] = value
		}
		jsonData = append(jsonData, item)
	}

	return json.Marshal(jsonData)
}
