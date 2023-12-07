package dbmanager

import (
	"SQL_Splitter/util"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/xwb1989/sqlparser"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// MySQL数据库的连接详情
type SqlAddress struct {
	Site_name string
	UserName  string
	Password  string
	Ip        string
	Port      string
	DbName    string
}

// 数据库表，包括表名、模式和关联的站点
type Table struct {
	Name  string   `json:"name"`
	Mode  string   `json:"mode"`
	Sites []string `json:"sites"`
}

// 管理数据库连接和表配置信息。
type DBM struct {
	Databases map[string]*sql.DB // 站点名字到数据库的映射
	etcdbar   *clientv3.Client   //

	tables map[string]Table // 表名字到数据库表的映射
	addrs  []SqlAddress     //站点名字到站点地址的映射
}

// 从JSON文件中读取表配置信息，并将其填充到tables映射中。
func (dbmp *DBM) init_conf() {

	//读取tabels.json
	file, err := os.Open(util.Conf_path + "tables.json")
	if err != nil {
		panic(err)
	}
	defer file.Close() // 确保在函数执行结束时关闭文件
	byteValue, _ := ioutil.ReadAll(file)
	var tables []Table
	json.Unmarshal(byteValue, &tables)
	for _, x := range tables {
		dbmp.tables[x.Name] = x
	}

	//读取global.json
	fileG, errG := os.Open(util.Conf_path + "global.json")
	if errG != nil {
		panic(errG)
	}
	defer fileG.Close()

	byteValue, _ = ioutil.ReadAll(fileG)

	json.Unmarshal(byteValue, &dbmp.addrs)

}

// 初始化一个etcd客户端连接。
func (dbmp *DBM) init_etcd() {
	//TODO 这里啥也没干
	if util.Test {
		return
	}
	cfg := clientv3.Config{
		Endpoints:   []string{util.EtcdAddr}, //etcd服务器的地址
		DialTimeout: 5 * time.Second,         //建立连接的超时时间
	}
	// 创建etcd客户端
	cli, err := clientv3.New(cfg)
	if err != nil {
		fmt.Printf("创建etcd客户端失败：%v \n", err)
		return
	} else {
		fmt.Println("连接etcd成功! (" + util.EtcdAddr + ")")
	}

	dbmp.etcdbar = cli
}

// 基于提供的SqlAddress实例初始化MySQL数据库连接，并将其存储在Databases映射中。
func (dbmp *DBM) init_mysql() {

	for _, addr := range dbmp.addrs {
		db, err := initDB(addr)
		if err != nil {
			fmt.Println(err)
			return
		}
		dbmp.Databases[addr.Site_name] = db
	}
}

func (dbmp *DBM) init() {
	dbmp.tables = make(map[string]Table)
	dbmp.Databases = make(map[string]*sql.DB)
	dbmp.init_conf()
	dbmp.init_etcd()
	dbmp.init_mysql()

}

// 解析输入的SQL字符串，确定其类型（当前仅处理SELECT语句），并将执行委托给特定的方法
func (dbmp *DBM) Do(sql_s string) {
	class_code := sqlparser.Preview(sql_s)
	if sqlparser.StmtType(class_code) == "SELECT" {
		dbmp.Select(sql_s)
	} else if sqlparser.StmtType(class_code) == "INSERT" {
		// TODO
		//dbmp.Insert(sql_s)
	}

}

func New_DBM() *DBM {
	var dbs DBM
	dbmp := &dbs
	dbmp.init()
	return dbmp
}
