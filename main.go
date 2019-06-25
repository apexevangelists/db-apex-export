package main

// DB-APEX-Export

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/howeyc/gopass"
	"github.com/juju/loggo"
	"github.com/spf13/viper"
	"gopkg.in/goracle.v2"
)

// TConfig - parameters in config file
type TConfig struct {
	configFile       string
	debugMode        bool
	appID            string
	outputFilename   string
	connectionConfig string
	connectionsDir   string
}

// TConnection - parameters passed by the user
type TConnection struct {
	dbConnectionString string
	username           string
	password           string
	hostname           string
	port               int
	service            string
}

var config = new(TConfig)
var connection TConnection
var programName = os.Args[0]

var logger = loggo.GetLogger(programName)

/********************************************************************************/
func setDebug(debugMode bool) {
	if debugMode == true {

		loggo.ConfigureLoggers(fmt.Sprintf("%s=DEBUG", programName))
		logger.Debugf("Debug log enabled")
	}
}

/********************************************************************************/
func parseFlags() {

	flag.StringVar(&config.configFile, "configFile", "config", "Configuration file for general parameters")
	flag.BoolVar(&config.debugMode, "debug", false, "Debug mode (default=false)")
	flag.StringVar(&config.appID, "appId", "", "Application ID to export (specify multiple seperated by a comma)")
	flag.StringVar(&config.outputFilename, "o", "", "Filename used for the export file (specify multiple seperated by a comma)")

	flag.StringVar(&config.connectionConfig, "connection", "", "Confguration file for connection")

	flag.StringVar(&connection.dbConnectionString, "db", "", "Database Connection, e.g. user/password@host:port/sid")

	flag.Parse()

	// At a minimum we either need a dbConnection or a configFile
	if (config.configFile == "") && (connection.dbConnectionString == "") {
		flag.PrintDefaults()
		os.Exit(1)
	}

}

/********************************************************************************/
func getPassword() []byte {
	fmt.Printf("Password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		// Handle gopass.ErrInterrupted or getch() read error
	}

	return pass
}

/********************************************************************************/
func loadConfig(configFile string) {
	if config.configFile == "" {
		return
	}

	logger.Debugf("reading configFile: %s", configFile)
	viper.SetConfigType("yaml")
	viper.SetConfigName(configFile)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	// need to set debug mode if it's not already set
	setDebug(viper.GetBool("debugMode"))

	config.connectionsDir = viper.GetString("connectionsDir")
	config.connectionConfig = viper.GetString("connectionConfig")

	config.debugMode = viper.GetBool("debugMode")

	if (viper.GetString("appID") != "") && (config.appID == "") {
		logger.Debugf("loadConfig: appID loaded: %s\n", viper.GetString("appID"))
		config.appID = viper.GetString("appID")
	}
	config.configFile = configFile
}

/********************************************************************************/
func loadConnection(connectionFile string) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName(config.connectionConfig)
	v.AddConfigPath(config.connectionsDir)

	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	if v.GetString("dbConnectionString") != "" {
		connection.dbConnectionString = v.GetString("dbConnectionString")
	}
	connection.username = v.GetString("username")
	connection.password = v.GetString("password")
	connection.hostname = v.GetString("hostname")
	connection.port = v.GetInt("port")
	connection.service = v.GetString("service")

	if (viper.GetString("appID") != "") && (config.appID == "") {
		logger.Debugf("loadConnection: export loaded: %s\n", v.GetString("export"))
		config.appID = v.GetString("appID")
	}

}

/********************************************************************************/
func getConnectionString(connection TConnection) string {
	var str = fmt.Sprintf("%s/%s@%s:%d/%s", connection.username,
		connection.password,
		connection.hostname,
		connection.port,
		connection.service)

	return str
}

/********************************************************************************/
// To execute, at a minimum we need (connection && (object || sql))
func checkMinFlags() {
	// connection is required
	bHaveConnection := (getConnectionString(connection) != "")

	// check if we have an appID
	bHaveObject := (config.appID != "")

	if !bHaveConnection || !bHaveObject {
		fmt.Printf("%s:\n", os.Args[0])
	}

	if !bHaveConnection {
		fmt.Printf("  requires a DB connection to be specified\n")
	}

	if !bHaveObject {
		fmt.Printf("  requires at least 1 application id to export\n")
	}

	if !bHaveConnection || !bHaveObject {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

/********************************************************************************/
func debugConfig() {
	logger.Debugf("config.configFile: %s\n", config.configFile)
	logger.Debugf("config.debugMode: %s\n", strconv.FormatBool(config.debugMode))
	logger.Debugf("config.appId: %s\n", config.appID)
	logger.Debugf("config.connectionConfig: %s\n", config.connectionConfig)
	logger.Debugf("connection.dbConnectionString: %s\n", connection.dbConnectionString)
}

/********************************************************************************/
// ReadAt reads at most len(p) bytes into p at offset.
/*
func (dl *DirectLob) ReadAt(p []byte, offset int64) (int, error) {
	n := C.uint64_t(len(p))
	n = n * 4
	if C.dpiLob_readBytes(dl.dpiLob, C.uint64_t(offset)+1, n, (*C.char)(unsafe.Pointer(&p[0])), &n) == C.DPI_FAILURE {
		return int(n), errors.Wrap(dl.conn.getError(), "readBytes")
	}
	return int(n), nil
}
*/

/********************************************************************************/
func exportApplication(db *sql.DB, appID string, outputFile string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// To have a valid LOB locator, we have to keep the Stmt around.
	lSQL := `declare tmp clob;
             begin
               dbms_lob.createtemporary(tmp, TRUE, DBMS_LOB.SESSION);
               :1 := tmp;
             end;`

	stmt, err := db.PrepareContext(ctx, lSQL)
	if err != nil {
		fmt.Printf("1: %s", err)
	}
	defer stmt.Close()
	var tmp goracle.Lob

	var fileName string
	var appAlias string
	var appWorkspace string

	tmp.IsClob = true

	if _, err := stmt.ExecContext(ctx, goracle.LobAsReader(), sql.Out{Dest: &tmp}); err != nil {
		fmt.Printf("Failed to create temporary lob: %+v", err)
	}

	/*p_split                   in boolean  default false,
	     p_with_date               in boolean  default false,
	     p_with_ir_public_reports  in boolean  default false,
	     p_with_ir_private_reports in boolean  default false,
	     p_with_ir_notifications   in boolean  default false,
	     p_with_translations       in boolean  default false,
	     p_with_pkg_app_mapping    in boolean  default false,
	     p_with_original_ids       in boolean  default false,
	     p_with_no_subscriptions   in boolean  default false,
	     p_with_comments           in boolean  default false,
	     p_with_supporting_objects in varchar2 default null,
		 p_with_acl_assignments    in boolean  default false
	*/

	lSQL = `declare      
               l_files     apex_t_export_files;
			   l_alias     varchar2(255);
			   l_workspace varchar2(255);
             begin                 
               -- get the alias
			   select a.workspace,
			          a.alias
				 into l_workspace,
				      l_alias
                 from apex_applications a
               where a.application_id = :1;

			   l_files := apex_export.get_application(p_application_id => :1);                                         
               :2 := l_files(1).contents;                          
               :3 := l_files(1).name;     
			   :4 := l_alias; 
			   :5 := l_workspace;
            end;`

	if _, err := db.ExecContext(ctx,
		lSQL,
		appID,
		tmp,
		sql.Out{Dest: &fileName},
		sql.Out{Dest: &appAlias},
		sql.Out{Dest: &appWorkspace},
	); err != nil {
		fmt.Printf("Failed to export: %s\n", err)
		os.Exit(1)
	}

	dl, err := tmp.Hijack()

	length, err := dl.Size()
	if err != nil {
		fmt.Printf("%s", err)
	}

	logger.Debugf("fileName: %s\n", fileName)
	logger.Debugf("fileSize: %d (bytes)\n", length)
	logger.Debugf("appAlias: %s\n", appAlias)

	// Allocate a byte array to receive the CLOB
	// note it needs to be 4*length to handle UTF8
	logger.Debugf("length: %d\n", length)
	buf := make([]byte, length)
	logger.Debugf("len(buf): %d\n", len(buf))

	_, err = dl.ReadAt(buf, 0)
	if err != nil {
		fmt.Printf("ReadAt Error: %s\n", err)
	}

	defer dl.Close()

	data := make([]byte, length)
	copy(data[:], buf[0:length])

	outputFileName := fmt.Sprintf("%s_%s_%s.sql", strings.ToLower(appWorkspace), outputFile, strings.ToLower(appAlias))

	err = ioutil.WriteFile(outputFileName, data, 0644)
	if err != nil {
		fmt.Printf("ioutil.WriteFile: %s", err)
	} else {
		logger.Debugf("Wrote %d bytes to file %s\n", length, outputFileName)
	}

}

/********************************************************************************/
func main() {

	parseFlags()
	setDebug(config.debugMode)
	loadConfig(config.configFile)
	loadConnection(config.connectionConfig)

	debugConfig()
	checkMinFlags()

	if connection.password == "" {
		connection.password = string(getPassword())
	}

	db, err := sql.Open("goracle", getConnectionString(connection))

	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// see if need to loop round the application ids
	apps := strings.Split(config.appID, ",")
	outputFiles := strings.Split(config.outputFilename, ",")

	for i, app := range apps {
		logger.Debugf("Exporting application [%d / %d]: %s\n", i+1, len(apps), app)

		logger.Debugf("outputFiles: %+v\n", outputFiles)
		logger.Debugf("len(outputFiles): %d\n", len(outputFiles))

		if len(outputFiles) == 1 {
			logger.Debugf("1\n")
			exportApplication(db, app, fmt.Sprintf("f%s", app))
		} else {
			logger.Debugf("2\n")
			exportApplication(db, app, outputFiles[i])
		}
	}

}
