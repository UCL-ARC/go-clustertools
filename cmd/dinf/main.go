package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/UCL-ARC/go-clustertools/internal/adhelper"
	"github.com/alecthomas/kingpin/v2"
	"github.com/go-ldap/ldap/v3"
)

var defaultPasswordFiles = []string{os.Getenv("DINF_ADPWFILE"), os.Getenv("HOME") + "/.adpw", "/shared/ucl/etc/adpw"}

var description = `
A program to quickly query Active Directory for user and group data.
`

var (
	app = kingpin.New("dinf", description)

	insecure    = app.Flag("insecure", "Insecurely ignore server certificate.").Short('i').Short('k').Bool()
	searchField = app.Flag("search-field", "Field to search on. (e.g. mail, memberOf, sn)").Short('f').Default("CN").String()

	ldapUrl       = app.Flag("server", "URL of the LDAP/AD server to connect to.").Short('s').Default("ldaps://ldap-auth-ad-slb.ucl.ac.uk:636/").String()
	bindUser      = app.Flag("user", "User to authenticate to the server with. (\"Bind\" user.)").Short('u').Default(`AD\sa-ritsldap01`).String()
	bindPassword  = app.Flag("password", "Password to authenticate to the server with. (\"Bind\" password.) If empty or not provided, files will be used.").Short('p').Default("").String()
	bindCredsFile = app.Flag("creds-file", "File to get bind credentials from. (Default: first of $DINF_ADPWFILE ~/.adpw /shared/ucl/etc/adpw)").Short('c').PlaceHolder("file").String()
	certFile      = app.Flag("cert", "Certificate to use with LDAPS. (Default: DINF_CERT)").PlaceHolder("file").Default("@").String() // @ is unset marker
	searchBase    = app.Flag("base", "Search base in the LDAP tree.").Short('b').Default("DC=ad,DC=ucl,DC=ac,DC=uk").String()
	returnFields  = app.Flag("output", "Command-separated fields to show in output. (Default: all)").Short('o').PlaceHolder("field[,field...]").String()

	bareVals  = app.Flag("bare", "Just print the values, without field names, no DN object labels.").Default("false").Bool()
	rawQuery  = app.Flag("raw", "Mandatory argument is a raw query, don't build a query yourself.").Default("false").Bool()
	quietMode = app.Flag("quiet", "Don't print output, just return 0 if there are results or 1 if not.").Default("false").Bool()

	searchTerm = app.Arg("search_term", "Search term (required).").Required().Strings()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var realPassword string

	if *bindPassword == "" {
		if *bindCredsFile != "" {
			realPassword = readPassword([]string{*bindCredsFile})
		} else {
			realPassword = readPassword(defaultPasswordFiles)
		}
	} else {
		realPassword = *bindPassword
	}

	// Override env var with cli arg, or set to nil if neither
	certFileEnv, certFileEnvIsSet := os.LookupEnv("DINF_CERTFILE")
	if *certFile == "@" {
		// @ here means it was not set by cli arg
		if certFileEnvIsSet {
			*certFile = certFileEnv
		} else {
			*certFile = ""
		}
	}

	ldapOpts := adhelper.LdapOpts{
		ServerUrl: *ldapUrl,
		Username:  *bindUser,
		Password:  realPassword,
		BaseDN:    *searchBase,
		Insecure:  *insecure,
		CertFile:  *certFile,
	}

	splitReturnFields := strings.Split(*returnFields, ",")
	if (len(splitReturnFields) == 1) && (splitReturnFields[0] == "") {
		splitReturnFields = []string{}
	}

	var searchExpression string
	if *rawQuery {
		searchExpression = (*searchTerm)[0]
	} else {
		searchExpression = fmt.Sprintf("(%s=%s)", *searchField, (*searchTerm)[0])
	}

	result, err := adhelper.RunADSearch(&ldapOpts, searchExpression, splitReturnFields)
	if err != nil {
		log.Fatal(err)
	}

	if *quietMode {
		if len(result.Entries) == 0 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	if len(result.Entries) == 0 {
		log.Fatalln("search result has no entries")
	}

	if *bareVals {
		BarePrint(result)
	} else {
		PrettierPrint(result, 2)
	}
}

// I abstracted the pretty-printing functions from the ldap package so I could alter the format.

// PrettierPrint outputs a human-readable description with indenting
func PrettierPrint(s *ldap.SearchResult, indent int) {
	for _, entry := range s.Entries {
		PrettierEPrint(entry, indent)
	}
}

// PrettierPrint outputs a human-readable description with indenting
func PrettierEAPrint(e *ldap.EntryAttribute, indent int) {
	for _, v := range e.Values {
		fmt.Printf("%s%s: %s\n", strings.Repeat(" ", indent), e.Name, filterPrintable(v))
	}
	//fmt.Printf("%s%s: %#v\n", strings.Repeat(" ", indent), e.Name, e.Values)
}

// PrettierPrint outputs a human-readable description indenting
func PrettierEPrint(e *ldap.Entry, indent int) {
	fmt.Printf("%sDN: %s\n", strings.Repeat(" ", indent), e.DN)
	for _, attr := range e.Attributes {
		PrettierEAPrint(attr, indent+2)
	}
}

// Bare value output, for scripting
func BarePrint(s *ldap.SearchResult) {
	for _, entry := range s.Entries {
		for _, attr := range entry.Attributes {
			for _, v := range attr.Values {
				fmt.Printf("%s\n", string(v))
			}
		}
	}
}
