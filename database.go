package main

import (
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type RequestLog struct {
	UserAgent string `bson:"user_agent"`
	JA3       string `bson:"ja3"`
	H2        string `bson:"h2"`
	PeetPrint string `bson:"peetprint"`
	IP        string `bson:"ip"`
	JsonAll   string `bson:"json_all"`
	Time      int64
}

type ByJA3 struct {
	JA3        string         `json:"ja3"`
	H2         map[string]int `json:"h2_fps"`
	PeetPrint  map[string]int `json:"peet_prints"`
	UserAgents map[string]int `json:"user_agents"`
}

type ByPeetPrint struct {
	PeetPrint  string         `json:"peet_print"`
	JA3        map[string]int `json:"ja3s"`
	H2         map[string]int `json:"h2_fps"`
	UserAgents map[string]int `json:"user_agents"`
}

type ByH2 struct {
	H2         string         `json:"h2_fp"`
	JA3        map[string]int `json:"ja3s"`
	PeetPrint  map[string]int `json:"peet_prints"`
	UserAgents map[string]int `json:"user_agents"`
}

type ByUserAgent struct {
	UserAgent string         `json:"useragent"`
	H2        map[string]int `json:"h2_fps"`
	JA3       map[string]int `json:"ja3s"`
	PeetPrint map[string]int `json:"peet_prints"`
}

func SaveRequest(req Response) {
	reqLog := RequestLog{
		JA3:       req.TLS.JA3,
		PeetPrint: req.TLS.PeetPrint,
		Time:      time.Now().Unix(),
	}

	if req.HTTPVersion == "h2" {
		reqLog.H2 = req.Http2.AkamaiFingerprint
	} else if req.HTTPVersion == "http/1.1" {
		reqLog.H2 = "-"
	}
	if c.LogIPs {
		parts := strings.Split(req.IP, ":")
		ip := strings.Join(parts[0:len(parts)-1], ":")
		reqLog.IP = ip
	}

	userAgent := GetUserAgent(req)
	lowerUserAgent := strings.ToLower(userAgent)
	// List of prohibited substrings in lowercase
	prohibitedSubstrings := []string{
		"curl", "telegram", "python", "go", "java", "php", "node",
		"wget", "ruby", "perl", "c++", "swift", "kotlin", "rust",
	}

	// Check if lowerUserAgent contains any prohibited substring
	for _, substring := range prohibitedSubstrings {
		if strings.Contains(lowerUserAgent, "okhttp") {
			break
		}
		if lowerUserAgent == "" || strings.Contains(lowerUserAgent, substring) {
			return
		}
	}
	reqLog.UserAgent = userAgent
	reqLog.JsonAll = req.ToJson()

	// 使用聚合管道去重
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$user_agent"},
			{"uniqueId", bson.D{{"$first", "$_id"}}},
			{"userAgent", bson.D{{"$first", "$user_agent"}}},
			{"JA3", bson.D{{"$first", "$JA3"}}},
			{"H2", bson.D{{"$first", "$H2"}}},
			{"PeetPrint", bson.D{{"$first", "$PeetPrint"}}},
			{"IP", bson.D{{"$first", "$IP"}}},
			{"JsonAll", bson.D{{"$first", "$json_all"}}},
			{"Time", bson.D{{"$first", "$Time"}}},
		}}},
		{{"$match", bson.D{
			{"userAgent", userAgent}, // 匹配给定的 userAgent
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	var existingRecords []RequestLog

	if err := cursor.All(ctx, &existingRecords); err != nil {
		log.Fatal(err)
	}

	if len(existingRecords) > 0 {
		// 如果已存在相同 userAgent 的记录，不进行插入
		fmt.Println("Record already exists for UserAgent:", existingRecords[0].UserAgent)
	} else {
		// 如果不存在相同 userAgent 的记录，进行插入
		reqLog := RequestLog{
			UserAgent: userAgent,
			JA3:       "your_ja3_data",       // 设置你的 JA3 数据
			H2:        "your_h2_data",        // 设置你的 H2 数据
			PeetPrint: "your_peetprint_data", // 设置你的 PeetPrint 数据
			IP:        "your_ip_data",        // 设置你的 IP 数据
			JsonAll:   "your_json_data",      // 设置你的 JSON 数据
			Time:      1234567890,            // 设置时间戳
		}

		// 插入新记录
		_, err = collection.InsertOne(ctx, reqLog)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println("Record inserted successfully!")
		}
	}

	log.Printf("Saved request %s\n", userAgent)
}

func GetTotalRequestCount() int64 {
	itemCount, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println(err)
		return -1
	}
	return itemCount
}

func queryDB(query, val string) []RequestLog {
	dbRes := []RequestLog{}
	cur, err := collection.Find(ctx, bson.D{{Key: query, Value: val}})
	if err != nil {
		log.Println("Error quering data:", err)
		return dbRes
	}

	for cur.Next(ctx) {
		var b RequestLog
		err := cur.Decode(&b)
		if err != nil {
			log.Println("Error decoding:", err)
			return dbRes
		}
		dbRes = append(dbRes, b)
	}

	if err := cur.Err(); err != nil {
		log.Println("Error - cur.Err()", err)
		return dbRes
	}

	if cur.Close(ctx) != nil {
		log.Println("Could not close")
	}
	return dbRes
}

const COUNT = 10

func GetByJa3(val string) ByJA3 {
	res := ByJA3{
		JA3:        val,
		H2:         map[string]int{},
		PeetPrint:  map[string]int{},
		UserAgents: map[string]int{},
	}

	dbRes := queryDB("ja3", val)

	for _, r := range dbRes {
		if v, ok := res.H2[r.H2]; ok {
			res.H2[r.H2] = v + 1
		} else {
			res.H2[r.H2] = 1
		}

		if v, ok := res.PeetPrint[r.PeetPrint]; ok {
			res.PeetPrint[r.PeetPrint] = v + 1
		} else {
			res.PeetPrint[r.PeetPrint] = 1
		}

		if v, ok := res.UserAgents[r.UserAgent]; ok {
			res.UserAgents[r.UserAgent] = v + 1
		} else {
			res.UserAgents[r.UserAgent] = 1
		}
	}

	res.PeetPrint = sortByVal(res.PeetPrint, COUNT)
	res.H2 = sortByVal(res.H2, COUNT)
	res.UserAgents = sortByVal(res.UserAgents, COUNT)

	return res
}

func GetByH2(val string) ByH2 {
	res := ByH2{
		H2:         val,
		JA3:        map[string]int{},
		PeetPrint:  map[string]int{},
		UserAgents: map[string]int{},
	}

	dbRes := queryDB("h2", val)

	for _, r := range dbRes {
		if v, ok := res.JA3[r.JA3]; ok {
			res.JA3[r.JA3] = v + 1
		} else {
			res.JA3[r.JA3] = 1
		}

		if v, ok := res.PeetPrint[r.PeetPrint]; ok {
			res.PeetPrint[r.PeetPrint] = v + 1
		} else {
			res.PeetPrint[r.PeetPrint] = 1
		}

		if v, ok := res.UserAgents[r.UserAgent]; ok {
			res.UserAgents[r.UserAgent] = v + 1
		} else {
			res.UserAgents[r.UserAgent] = 1
		}
	}

	res.PeetPrint = sortByVal(res.PeetPrint, COUNT)
	res.JA3 = sortByVal(res.JA3, COUNT)
	res.UserAgents = sortByVal(res.UserAgents, COUNT)
	return res
}

func GetByPeetPrint(val string) ByPeetPrint {
	res := ByPeetPrint{
		PeetPrint:  val,
		H2:         map[string]int{},
		JA3:        map[string]int{},
		UserAgents: map[string]int{},
	}

	dbRes := queryDB("peetprint", val)

	for _, r := range dbRes {
		if v, ok := res.H2[r.H2]; ok {
			res.H2[r.H2] = v + 1
		} else {
			res.H2[r.H2] = 1
		}

		if v, ok := res.JA3[r.JA3]; ok {
			res.JA3[r.JA3] = v + 1
		} else {
			res.JA3[r.JA3] = 1
		}

		if v, ok := res.UserAgents[r.UserAgent]; ok {
			res.UserAgents[r.UserAgent] = v + 1
		} else {
			res.UserAgents[r.UserAgent] = 1
		}
	}
	res.JA3 = sortByVal(res.JA3, COUNT)
	res.H2 = sortByVal(res.H2, COUNT)
	res.UserAgents = sortByVal(res.UserAgents, COUNT)

	return res
}

func GetByUserAgent(val string) ByUserAgent {
	res := ByUserAgent{
		UserAgent: val,
		H2:        map[string]int{},
		JA3:       map[string]int{},
		PeetPrint: map[string]int{},
	}

	decodedValue, err := url.QueryUnescape(val)
	if err != nil {
		return res
	}
	fmt.Println(val)

	dbRes := queryDB("user_agent", decodedValue)

	for _, r := range dbRes {
		if v, ok := res.H2[r.H2]; ok {
			res.H2[r.H2] = v + 1
		} else {
			res.H2[r.H2] = 1
		}

		if v, ok := res.JA3[r.JA3]; ok {
			res.JA3[r.JA3] = v + 1
		} else {
			res.JA3[r.JA3] = 1
		}

		if v, ok := res.PeetPrint[r.PeetPrint]; ok {
			res.PeetPrint[r.PeetPrint] = v + 1
		} else {
			res.PeetPrint[r.PeetPrint] = 1
		}
	}
	res.JA3 = sortByVal(res.JA3, COUNT)
	res.H2 = sortByVal(res.H2, COUNT)
	res.PeetPrint = sortByVal(res.PeetPrint, COUNT)

	return res
}
