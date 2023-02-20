package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jf-tech/go-corelib/jsons"
	"github.com/spf13/cobra"

	"github.com/Perachi0405/ownediparse"
	"github.com/Perachi0405/ownediparse/transformctx"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Launches op into HTTP server mode with its REST APIs.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("invoke the doServer()")
			doServer()
		},
	}
	port int
)

func init() {
	// fmt.Println("Init in serverCmd.go")
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "the listening HTTP port")
}

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

func doServer() {
	// fmt.Println("Inside the doServer()")
	transformRouter := chi.NewRouter()
	transformRouter.Use(middleware.RealIP)
	transformRouter.Use(middleware.AllowContentType(contentTypeJSON))
	transformRouter.Post("/", httpPostTransform)

	samplesRouter := chi.NewRouter()
	samplesRouter.Get("/", httpGetSamples)

	versionRouter := chi.NewRouter()
	versionRouter.Get("/", httpGetVersion)

	rootRouter := chi.NewRouter()
	rootRouter.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.FileServer(http.Dir(filepath.Join(serverCmdDir(), "web"))).ServeHTTP(w, req)
	})
	rootRouter.Mount("/transform", transformRouter)
	rootRouter.Mount("/samples", samplesRouter)
	rootRouter.Mount("/version", versionRouter)

	envPort, found := os.LookupEnv("PORT")
	if found {
		var err error
		port, err = strconv.Atoi(envPort)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Listening on port %d ...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), rootRouter))
}

func serverCmdDir() string {
	_, filename, _, _ := runtime.Caller(1)
	absDir, _ := filepath.Abs(filepath.Dir(filename))
	fmt.Println("absDir", absDir)
	return absDir
}

func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
	log.Print(code)
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	writeError(w, msg, http.StatusBadRequest)
}

func writeInternalServerError(w http.ResponseWriter, msg string) {
	writeError(w, msg, http.StatusInternalServerError)
}

func writeSuccessJSON(w http.ResponseWriter, jsonStr string) {
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(jsons.BPJ(jsonStr)))
	log.Print(http.StatusOK)
}

//writing the success response
func writeSuccess(w http.ResponseWriter, v interface{}) {
	writeSuccessJSON(w, jsons.BPM(v))
}

//old logic
type reqTransform struct {
	Schema     string            `json:"schema"`
	Input      string            `json:"input"`
	Properties map[string]string `json:"properties"`
}

func httpPostTransform(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving POST '/transform' request from %s ... ", r.RemoteAddr)
	fmt.Println("Body checking", r.Body)
	fmt.Println("Type of body", reflect.TypeOf(r.Body))

	//normal logic
	b, err := ioutil.ReadAll(r.Body)
	var req reqTransform
	// Readedstr := string(b)
	// fmt.Println("Readedstr", Readedstr)

	//normal logic UI
	//err = json.Unmarshal(b, &req)
	//The issue is based on enveloping the data within the " " for the input file.
	//Tried with the sample input file by changing the data in single line - works but enclose them in "" -fails

	//Unmarshall using map logic
	var result map[string]interface{}
	err2 := json.Unmarshal(b, &result)
	if err2 != nil {
		fmt.Println("unmarshalleing 2 error", err2)
	}
	fmt.Println("Result from marshal", result)
	schemafrmmap := result["schema"]
	inputfrmmap := result["input"]
	// fmt.Println("schema from map", result["schema"])
	// var result1 map[string]interface{}
	// err3 := json.Unmarshal([]byte(inputfrmmap),&result1)
	fmt.Println("Length of the input array", inputfrmmap)

	schemaMarsh, err := JSONMarshal(schemafrmmap)
	if err != nil {
		fmt.Println("Schema marshalling error", err)
	}

	inputMarsh, err := JSONMarshal(inputfrmmap)
	if err != nil {
		fmt.Println("Input marshalling error", err)
	}
	var trimedprefixbyte []byte
	var trimedsufixbyte []byte
	trimedprefixbyte = bytes.TrimPrefix(inputMarsh, []byte{34})
	fmt.Println("Trimmed values prefix", trimedprefixbyte)
	trimedsufixbyte = bytes.TrimSuffix(trimedprefixbyte, []byte{34, 10})
	fmt.Println("Trimmed values suffix", trimedsufixbyte)
	// trimedsufixbyte :=bytes.HasSuffix()

	// fmt.Println("inputMarsh", inputMarsh)
	// str1 := bytes.NewBuffer(inputMarsh).String()
	// fmt.Println("Sting array of byte", str1[0])//34
	// fmt.Println("Marshal the input frm map", string(inputMarsh))

	req.Schema = string(schemaMarsh)
	req.Input = string(trimedsufixbyte)

	fmt.Println("Byte input array checking", []byte(req.Input)) //34

	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid request body. err: %s", err))
		return
	}
	s, err := ownediparse.NewSchema("test-schema", strings.NewReader(req.Schema)) // sends the name and the reader
	// fmt.Println("NewSchemaOutput", string(s.Content()))                          //validated Json schema in JSON format 'without \n special characters'
	//Getting the transformed data
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid schema. err: %s", err))
		return
	}

	// s.NewTransform(
	// 	"test-input", strings.NewReader(req.Input), &transformctx.Ctx{ExternalProperties: req.Properties})
	t, err := s.NewTransform("test-input", strings.NewReader(req.Input), &transformctx.Ctx{ExternalProperties: req.Properties}) // passing the name, Input file and properties with the schema s,
	//fmt.Println("newReader", strings.NewReader(req.Input)) //creates a new Reader
	fmt.Println("output of newTransform", t)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: unable to new transform. err: %s", err))
		return
	}
	var records []string
	for {
		b, err := t.Read()              //the Read func is called for many times
		fmt.Println("Reading t for", t) //unknown data in object format. TransformInstance &{0xc000deeec0 <nil> 0xc000098080}

		if err == io.EOF {
			fmt.Println("executes err EOF")
			break
		}
		if err != nil {
			writeBadRequest(w, fmt.Sprintf("bad request: transform failed. err: %s", err))
			return
		}
		// fmt.Println("Data type", b) //transformed Data in byte type
		records = append(records, string(b))
	}
	//fmt.Println("Transformed records", records)
	writeSuccessJSON(w, "["+strings.Join(records, ",")+"]") // return the parsed file to the UI
	// log.Print(jsons.BPM(req))
}

//old logic
// fmt.Println("type of reqschema", reflect.TypeOf(schemaval))
// fmt.Println("type of reqinput", reflect.TypeOf(reqinput))
// fmt.Println("request properties", req.Properties) //map[]
//err = json.Unmarshal(res, &req) //postman logic
// fmt.Println("request properties", req.Properties)
// err = json.Unmarshal(b, &result)

// fmt.Println("Result schema", result)
//second unmarshal
// err2 := json.Unmarshal(b, &result)

//Logic - 6

var (
	sampleDir                  = "../../extensions/v21/samples/"
	sampleFormats              = []string{"edi"}
	sampleInputFilenamePattern = regexp.MustCompile("^([0-9]+[_a-zA-Z0-9]+)\\.input\\.[a-z]+$")
)

//
type sample struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	Input  string `json:"input"`
}

//new Marshal replacement for json.Marshal
func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

//
func httpGetSamples(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving GET '/samples' request from %s ... ", r.RemoteAddr)
	samples := []sample{}
	//getting the files from the edi dir
	dir := filepath.Join(serverCmdDir(), sampleDir, "edi") //ownediparse/extensions/v21/samples/edi
	fmt.Println("Direc", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		goto getSampleFailure
	}
	// for _, format := range sampleFormats {
	// 	fmt.Println("format httpGetsamples", format)
	// 	dir := filepath.Join(serverCmdDir(), sampleDir, format) // dir : =filepath.join
	// 	files, err := ioutil.ReadDir(dir)
	// 	fmt.Println("dirName", dir)
	// 	fmt.Println("Filename", files) //getting the files from each folder
	// 	if err != nil {
	// 		goto getSampleFailure
	// 	}
	for _, f := range files {
		submatch := sampleInputFilenamePattern.FindStringSubmatch(f.Name()) //make a filename from the schema name
		// fmt.Println("submatch", submatch)
		if len(submatch) < 2 {
			continue
		}
		sample := sample{
			Name: filepath.Join("edi", submatch[1]),
		} //Schema name
		// fmt.Println("sample name", sample)
		schema, err := ioutil.ReadFile(filepath.Join(dir, submatch[1]+".schema.json")) //Reading the schema
		// fmt.Println("Schema readed", schema)                                           // read the schemas in byte array format.
		if err != nil {
			goto getSampleFailure
		}
		sample.Schema = string(schema) // give values to the schema key
		input, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			goto getSampleFailure
		}
		sample.Input = string(input)      // give values to the input key
		samples = append(samples, sample) // return the response array of names
	}
	// }
	writeSuccess(w, samples) //Return the 200
	return

getSampleFailure:
	writeInternalServerError(w, "unable to get samples")
	return
}

//invoked at first
func httpGetVersion(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Executing the ownediparser solution......")
	log.Printf("Serving GET '/version' request from %s ... ", r.RemoteAddr)
	writeSuccess(w, build)
}
