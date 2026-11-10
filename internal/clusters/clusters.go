package clusters

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

// Old Legion is here for completeness and in case it needs to be queried.
// In practice, it probably won't be used much.
var clusterAccountingDBs = map[string]string{
	"myriad":   "myriad_sgelogs",
	"legion":   "legion_sgelogs",
	"grace":    "grace_sgelogs",
	"thomas":   "thomas_sgelogs",
	"michael":  "michael_sgelogs",
	"kathleen": "kathleen_sgelogs",
	"young":    "young_sgelogs", // I hate this
	"legion1":  "sgelogs",
	"legion2":  "sgelogs2",
	"legion3":  "legion_sgelogs", // Currently a synonym for "legion"
}

// We used to use regexes to work out which cluster the current hostname was from,
//
//	but since the namespace collapse of Legion from login{05..09} to login{01..02},
//	hostname can no longer be reliably used to find cluster name.
func GetClusterNameFromSGEIdent() (string, error) {
	clusterNameBytes, err := os.ReadFile("/opt/sge/default/common/cluster_name")
	if err != nil {
		return "", errors.New("could not get cluster name: " + err.Error())
	}

	// The ident file has a trailing newline
	clusterName := strings.TrimSuffix(string(clusterNameBytes), "\n")

	return clusterName, nil
}

// SGE ident used to be good enough to get cluster name, but now we have
//
//	Slurm clusters as well, with the *same name*, so we need a better source
func GetClusterNameFromResolvConf() (string, error) {
	resolvBytes, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return "", errors.New("could not get cluster name from DNS resolver config: " + err.Error())
	}

	// Could remove the ucl.ac.uk requirement here and make it work locally?
	// Also, in theory you can give the search directive multiple domains
	//  but we never do so hopefully it's fine
	reClusterName := regexp.MustCompile(`(?m)^[ \t]*search[ \t]+([a-z0-9A-Z.-]+)\.ucl\.ac\.uk\.?[ \t]*$`)

	submatches := reClusterName.FindAllSubmatch(resolvBytes, -1)

	if submatches == nil {
		return "", errors.New("could not find cluster name in DNS resolver config")
	}

	// Only the last search directive in the config is used (per the man page)
	lastMatchIndex := len(submatches) - 1

	return string(submatches[lastMatchIndex][1]), nil
}

// Wrapper function so that caller does not need to know what method is used.
func GetLocalClusterName() (string, error) {
	clusterName, err := GetClusterNameFromResolvConf()
	if err != nil {
		return "", err
	}
	return clusterName, nil
}

func GetClusterAccountingDBName(clusterName string) (string, error) {
	dbName, wasPresent := clusterAccountingDBs[clusterName]
	if wasPresent == false {
		return "", errors.New("This cluster [" + clusterName + "] does not have a known associated accounting DB.")
	}
	return dbName, nil
}

func GetLocalClusterAccountingDBName() (string, error) {
	clusterName, err := GetLocalClusterName()
	if err != nil {
		return "", err
	}
	return GetClusterAccountingDBName(clusterName)
}
