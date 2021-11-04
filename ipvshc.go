package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// structure for config
type config struct {
	id       int
	host     string
	thold    int
	interval int
	tgtoken  string
	hostname string
}

// structure for healthcheck
type check struct {
	id      int
	vs      string
	vaddr   string
	raddr   string
	caddr   string
	path    string
	mode    string
	weight  string
	tgid    string
	status  string
	changed string
}

// function main
func main() {

	//function get config
	cf := loadConfig()
	//fmt.Println(cf.id, cf.host, cf.thold, cf.interval, cf.tgtoken)

	//function get healthchecks
	checks := loadChecks()

	// wait for anding all healthchecks
	var wg sync.WaitGroup
	wg.Add(len(checks)) // get len
	//fmt.Println(len(checks))

	// loop for healthchek
	for _, hc := range checks {
		//fmt.Println(hc.id, hc.vs, hc.vaddr, hc.raddr, hc.caddr, hc.path, hc.mode, hc.weight, hc.tgid, hc.status, hc.changed) // print conf for debug
		go healthcheck(hc, cf, &wg) // parallel go healthcheck by gorutine
	}
	wg.Wait() // wait for anding all healthchecks
}

// function  for get config
func loadConfig() config {

	db, err := sql.Open("sqlite3", "/opt/ipvshc/ipvshc.db") // open connect to DB
	if err != nil {
		panic(err)
	}
	defer db.Close()
	row, err := db.Query("select config.id, config.host, config.thold, config.interval, config.tgtoken from config limit 1;") // select config
	if err != nil {
		panic(err)
	}
	defer row.Close()
	var cf config
	row.Next()
	err = row.Scan(&cf.id, &cf.host, &cf.thold, &cf.interval, &cf.tgtoken) // get
	if err != nil {
		fmt.Println(err)
	}
	hostname, err := os.Hostname() //get hostname from OS
	if err != nil {
		panic(err)
	}
	//fmt.Println("hostname:", hostname)
	cf.hostname = hostname

	//fmt.Println(cf.id, cf.host, cf.thold, cf.interval, cf.tgtoken) // show all
	fmt.Println("host: "+cf.hostname, "src:", cf.host, "thold:", cf.thold, "interval:", cf.interval, "sec") // show param
	return cf
}

//function for get all healthchecks
func loadChecks() []check {

	db, err := sql.Open("sqlite3", "/opt/ipvshc/ipvshc.db") // connect to DB
	if err != nil {
		panic(err)
	}
	defer db.Close()
	//rows, err := db.Query("SELECT healthcheck.*, state.status, MAX(state.changed) FROM healthcheck LEFT JOIN state ON healthcheck.raddr=state.raddr GROUP BY healthcheck.id;") // select HC
	rows, err := db.Query("SELECT healthcheck.id, healthcheck.vs, healthcheck.vaddr, healthcheck.raddr, healthcheck.caddr, healthcheck.path, healthcheck.mode, healthcheck.weight, healthcheck.tgid, state.status, MAX(state.changed) FROM healthcheck LEFT JOIN state ON healthcheck.raddr=state.raddr WHERE healthcheck.state='enable' GROUP BY healthcheck.id ;") // select HC
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	checks := []check{}

	for rows.Next() {
		hc := check{}
		err := rows.Scan(&hc.id, &hc.vs, &hc.vaddr, &hc.raddr, &hc.caddr, &hc.path, &hc.mode, &hc.weight, &hc.tgid, &hc.status, &hc.changed) //get
		if err != nil {
			fmt.Println(err)
			continue
		}
		checks = append(checks, hc)
	}
	return checks
}

// function for check instances
func healthcheck(hc check, cf config, wg *sync.WaitGroup) { // add parametr hc, cf, wg
	defer wg.Done()

	thold := cf.thold // update thold

	for ; thold > 0; thold-- { //loop thold

		// Send http Check IPVS instance
		ok := true
		// /usr/bin/curl hc.caddr/hc.path --interface cf.host --connect-timeout 5 -k -s -f -o /dev/null && echo 'SUCCESS' || echo 'ERROR'
		resp := exec.Command("/usr/bin/curl", hc.caddr+"/"+hc.path, "--interface", cf.host, "--connect-timeout", "5", "-k", "-s", "-f")
		_, err := resp.Output()
		if err != nil {
			//fmt.Println(err.Error())
			ok = false
		}
		//fmt.Println(ok)

		// Send http Check IPVS instance ( defferent version)
		// resp, err := http.Get("http://" + hc.caddr + "/" + hc.path)
		// if err != nil {
		// 	//fmt.Println("error", err, resp)
		// 	ok = false
		// }
		// if ok { // condition  result
		// 	defer resp.Body.Close()
		// 	//fmt.Println(resp.StatusCode)
		// 	if resp.StatusCode != 200 {
		// 		ok = false
		// 	}
		// }

		// conditions:
		if ok && hc.status == "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: PASS STILL", hc.raddr)
			break
		} else if ok && hc.status != "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: PASS and ADD", hc.raddr)

			// set IPVS instance
			cmd := exec.Command("/usr/sbin/ipvsadm", "-e", "-"+hc.vs, hc.vaddr, "-r", hc.raddr, "-"+hc.mode, "-w", hc.weight)
			stdout, err := cmd.Output()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println(string(stdout)) // Print the output

			// write to DB status
			db, err := sql.Open("sqlite3", "/opt/ipvshc/ipvshc.db") // connect to DB
			if err != nil {
				panic(err)
			}
			defer db.Close()
			//result, err := db.Exec("INSERT INTO state (raddr, status) VALUES ('" + hc.raddr + "', 'OK'); DELETE FROM state WHERE id IN (SELECT id FROM state WHERE raddr='" + hc.raddr + "' ORDER BY id DESC LIMIT 10 OFFSET 10);") //  insert  OK status and remove oldest status then 10 last
			_, err = db.Exec("INSERT INTO state (raddr, status) VALUES ('" + hc.raddr + "', 'OK'); DELETE FROM state WHERE id IN (SELECT id FROM state WHERE raddr='" + hc.raddr + "' ORDER BY id DESC LIMIT 10 OFFSET 10);") //  insert  OK status and remove oldest status then 10 last
			if err != nil {
				panic(err)
			}
			//fmt.Println(result.LastInsertId()) // id last add object
			//fmt.Println(result.RowsAffected()) // count add string

			// send alert
			cmdcurl := exec.Command("/usr/bin/curl", "-s", "-X", "POST", "https://api.telegram.org/bot"+cf.tgtoken+"/sendMessage", "-d", "chat_id="+hc.tgid, "-d", "text= ✅ "+cf.hostname+": src ip "+cf.host+":"+" LB "+hc.raddr+" changed state to UP")
			_, err = cmdcurl.Output()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			// send alert ( defferent version)
			// _, alertErr := http.PostForm(
			// 	"https://api.telegram.org/bot"+cf.tgtoken+"/sendMessage",
			// 	url.Values{
			// 		"chat_id": {hc.tgid},
			// 		"text":    {" ✅ " + cf.hostname + "|" + cf.host + ":" + " LB " + hc.raddr + " changed state to DOWN"},
			// 	},
			// )
			// if err != nil {
			// 	fmt.Println(alertErr.Error())
			// 	return
			// }

			break
		} else if thold != 1 && hc.status == "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: FAIL", hc.raddr)
		} else if thold != 1 && hc.status != "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: FAIL STILL", hc.raddr)
		} else if thold == 1 && hc.status != "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: FAIL STILL... ", hc.raddr)
		} else if thold == 1 && hc.status == "OK" {
			fmt.Println(cf.host, "IPVSHealthCheck: FAIL and DELETE", hc.raddr)

			// set IPVS instance
			cmd := exec.Command("/usr/sbin/ipvsadm", "-e", "-"+hc.vs, hc.vaddr, "-r", hc.raddr, "-"+hc.mode, "-w 0")
			stdout, err := cmd.Output()

			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println(string(stdout)) // Print the output

			// write to DB status
			db, err := sql.Open("sqlite3", "/opt/ipvshc/ipvshc.db") // connect to DB
			if err != nil {
				panic(err)
			}
			defer db.Close()
			//result, err := db.Exec("INSERT INTO state (raddr, status) VALUES ('" + hc.raddr + "', 'ERROR'); DELETE FROM state WHERE id IN (SELECT id FROM state WHERE raddr='" + hc.raddr + "' ORDER BY id DESC LIMIT 10 OFFSET 10);") // insert ERROR status and remove oldest status then 10 last
			_, err = db.Exec("INSERT INTO state (raddr, status) VALUES ('" + hc.raddr + "', 'ERROR'); DELETE FROM state WHERE id IN (SELECT id FROM state WHERE raddr='" + hc.raddr + "' ORDER BY id DESC LIMIT 10 OFFSET 10);") // insert ERROR status and remove oldest status then 10 last
			if err != nil {
				panic(err)
			}
			//fmt.Println(result.LastInsertId()) // id last add object
			//fmt.Println(result.RowsAffected()) // count add string

			//send alert
			cmdcurl := exec.Command("/usr/bin/curl", "-s", "-X", "POST", "https://api.telegram.org/bot"+cf.tgtoken+"/sendMessage", "-d", "chat_id="+hc.tgid, "-d", "text= ⚠️️ "+cf.hostname+": src ip "+cf.host+":"+" LB "+hc.raddr+" changed state to DOWN")
			_, err = cmdcurl.Output()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			// send alert ( defferent version)
			// _, alertErr := http.PostForm(
			// 	"https://api.telegram.org/bot"+cf.tgtoken+"/sendMessage",
			// 	url.Values{
			// 		"chat_id": {hc.tgid},
			// 		"text":    {"⚠️️" + cf.hostname + "|" + cf.host + ":" + " LB " + hc.raddr + " changed state to DOWN"},
			// 	},
			// )
			// if err != nil {
			// 	fmt.Println(alertErr.Error())
			// 	return
			// }

		}
		if thold > 1 { // for last loop exit
			time.Sleep(time.Duration(cf.interval) * time.Second)
		}
	}
}
