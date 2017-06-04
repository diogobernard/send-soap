package main

// Interrogations du serveur SOAP de MH
// voir http://www.mountyhall.com/Forum/display_topic_threads.php?ThreadID=2171938

import (
	"bytes"
	//"chrall"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"os"
	"bufio"
	"encoding/base64"
)

//const MH_SOAP_URL = "http://sp.mountyhall.com/SP_WebService.php"

// params : numero (%d), mdprestreint (%s)
// params : numero (%d), mdprestreint (%s)
const SOAP_PROFIL_QUERY_FORMAT = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:tns=\"urn:SP_WebService\" xmlns:soap=\"http://schemas.xmlsoap.org/wsdl/soap/\" xmlns:wsdl=\"http://schemas.xmlsoap.org/wsdl/\" xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" ><SOAP-ENV:Body><mns:Profil xmlns:mns=\"uri:mhSp\" SOAP-ENV:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\"><numero xsi:type=\"xsd:string\">%d</numero><mdp xsi:type=\"xsd:string\">%s</mdp></mns:Profil></SOAP-ENV:Body></SOAP-ENV:Envelope>"
//const SOAP_CHARACTERISTIQUES_QUERY_FORMAT = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:tns=\"urn:SP_WebService\" xmlns:soap=\"http://schemas.xmlsoap.org/wsdl/soap/\" xmlns:wsdl=\"http://schemas.xmlsoap.org/wsdl/\" xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" ><SOAP-ENV:Body><mns:Caracteristiques xmlns:mns=\"uri:mhSp\" SOAP-ENV:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\"><numero xsi:type=\"xsd:string\">%d</numero><mdp xsi:type=\"xsd:string\">%s</mdp></mns:Caracteristiques></SOAP-ENV:Body></SOAP-ENV:Envelope>"
//const SOAP_VUE_QUERY_FORMAT = "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:tns=\"urn:SP_WebService\" xmlns:soap=\"http://schemas.xmlsoap.org/wsdl/soap/\" xmlns:wsdl=\"http://schemas.xmlsoap.org/wsdl/\" xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" ><SOAP-ENV:Body><mns:Vue xmlns:mns=\"uri:mhSp\" SOAP-ENV:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\"><numero xsi:type=\"xsd:string\">%d</numero><mdp xsi:type=\"xsd:string\">%s</mdp></mns:Vue></SOAP-ENV:Body></SOAP-ENV:Envelope>"


// The URL of the SOAP server
const MH_SOAP_URL = "http://localhost:8080/HubInterfacesSoap/soap/IHubInterfaces"

// this is just the message I'll send for interrogation, with placeholders
//  for my parameters
const SOAP_VUE_QUERY_FORMAT = `<soapenv:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:urn="urn:HubInterfacesIntf-IHubInterfaces">
<soapenv:Header/>
<soapenv:Body>
<urn:importaArquivo soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<aPin xsi:type="xsd:string">?</aPin>
<aArquivo xsi:type="xsd:string">?</aArquivo>
</urn:importaArquivo>
</soapenv:Body>
</soapenv:Envelope>`

type SoapItem struct { // un objet vu
	Numero    int
	Nom       string
	Type      string
	PositionX int
	PositionY int
	PositionN int
	Monde     int
}

type SoapVue struct {
	Items []SoapItem "return>item"
}

type SoapProfil struct { // un peu incomplet, certes
	Numero int
}
type SoapFault struct {
	Faultstring string
	Detail      string
}
type SoapBody struct {
	Fault          SoapFault
	ProfilResponse SoapProfil
	VueResponse    SoapVue
}
type SoapEnvelope struct {
	XMLName xml.Name
	Body    SoapBody
}

/*
Structure d'un message d'erreur:
{{http://schemas.xmlsoap.org/soap/envelope/ Body} []}
{{http://schemas.xmlsoap.org/soap/envelope/ Fault} []}
{{ faultcode} [{{http://www.w3.org/2001/XMLSchema-instance type} xsd:string}]}
SERVER
{{ faultcode}}
{{ faultactor} [{{http://www.w3.org/2001/XMLSchema-instance type} xsd:string}]}
{{ faultactor}}
{{ faultstring} [{{http://www.w3.org/2001/XMLSchema-instance type} xsd:string}]}
Erreur 2
{{ faultstring}}
{{ detail} [{{http://www.w3.org/2001/XMLSchema-instance type} xsd:string}]}
Troll inexistant
{{ detail}}
{{http://schemas.xmlsoap.org/soap/envelope/ Fault}}
{{http://schemas.xmlsoap.org/soap/envelope/ Body}}
{{http://schemas.xmlsoap.org/soap/envelope/ Envelope}}
*/

func DumpAll(r io.Reader) {
	b, e := ioutil.ReadAll(r)
	if e != nil {
		fmt.Println("Erreur lecture :")
		fmt.Println(e)
	}
	s := string(b)
	fmt.Print(s)
}

func main() {

	args := os.Args
	endPoint := args[1]
	pin := args[2]
	fileStr := args[3]

	fmt.Println("ENDPOINT: ",endPoint)
	fmt.Println("PIN: ",pin)
	fmt.Println("ARQUIVO: ",fileStr)

	file64 := buildFileInBase64(fileStr)

	query := `<soapenv:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:urn="urn:HubInterfacesIntf-IHubInterfaces">
<soapenv:Header/>
<soapenv:Body>
<urn:importaArquivo soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<aPin xsi:type="xsd:string">%s</aPin>
<aArquivo xsi:type="xsd:string">%s</aArquivo>
</urn:importaArquivo>
</soapenv:Body>
</soapenv:Envelope>`

	queryFmt := fmt.Sprintf(query,pin,file64)

	GetSoapEnvelope(endPoint,queryFmt)
}

func buildFileInBase64(fileStr string) string{
	file, errFile := os.Open(fileStr)

	if errFile != nil {
		fmt.Println(errFile)
		os.Exit(1)
	}

	defer file.Close()

	fInfo, _ := file.Stat()
	var size int64 = fInfo.Size()
	buf := make([]byte, size)

	// read file content into buffer
	fReader := bufio.NewReader(file)
	fReader.Read(buf)

	// convert the buffer bytes to base64 string - use buf.Bytes() for new image
	fileBase64Str := base64.StdEncoding.EncodeToString(buf)
	fmt.Println("Arquivo convertido para base64: ",fileBase64Str)
	return fileBase64Str
}

func GetSoapEnvelope(endpoint string, query string,) (envelope *SoapEnvelope) {
	fmt.Println("Mandando para o endereco: ",endpoint)
	fmt.Println("QUERY: ", query)

	httpClient := new(http.Client)
	soapRequestContent := fmt.Sprintf(query)

	fmt.Println("resposta: ", soapRequestContent)

	resp, err := httpClient.Post(endpoint, "text/xml; charset=utf-8", bytes.NewBufferString(soapRequestContent))

	fmt.Println("resposta: ", resp)

	if err != nil {
		fmt.Println("Erreur : " + err.Error())
		return nil
	}
	// là on fait du lourd : on passe par une chaine car j'ai pas le temps ce soir de trouver comment sauter directement un bout du flux jusqu'au début du xml
	b, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		fmt.Println("Erreur lecture :")
		fmt.Println(e)
	}
	in := string(b)
	indexDebutXml := strings.Index(in, "<?xml version")
	if indexDebutXml > 0 {
		fmt.Printf("Erreur message SOAP. Début XML à l'index %d\n", indexDebutXml)
		in = in[indexDebutXml:len(in)]
	}
	//fmt.Print(in)
	parser := xml.NewDecoder(bytes.NewBufferString(in))
	//parser.CharsetReader = chrall.CharsetReader
	envelope = new(SoapEnvelope)
	err = parser.DecodeElement(&envelope, nil)
	if err != nil {
		fmt.Println("Erreur au décodage du XML : " + err.Error())
		return nil
	}

	resp.Body.Close()

	return
}
