/*该程序用于实现固定资产随机盘查和最短路径的生成，主要实现了以下五个功能：
  1.可以通过该程序直接调用MongoDB软件并从中管理储存于其中的数据
  2.可以通过该程序导入一个CSV座位图，并更改和保存在MongoDB中各固定资产数据的坐标位置
  3.可以通过该程序输入抽查比例随机确定抽查的固定资产对象并显示
  4.确定随机抽查对象后，根据这几个对象的相对位置分别计算从每个点出发抽查的最短路径
  5.显示输出从每个点出发的最短路径
*/

package main

import (
	"bufio"        //用于使用缓存读入器，方便文件读入
	"encoding/csv" //方便对于CSV文件的读入
	"fmt"          //标准输入输出必要的包
	"io"           //文件末尾EOF的判断需要用到io包
	"math/rand"    //随机数的选取需要用到随机函数来生成
	"os"           //对于系统文件的打开、关闭和产生io.Reader型的文件指针
	"strconv"      //当从文件中把所有的信息以字符的形式读入后，需要将字符串转换为各种类型的信息
	"strings"      //由于从文件当中的读入是整行整行读入，所以需要用到string.split函数来把一行的多个数据分开
	"time"         //生成随机数要想每次都不同，最好的方式就是用一直在改变的时间作为基础

	"gopkg.in/mgo.v2"      //该包中含有使用golang对MongoDB操作的重要函数，包括连接该软件也是
	"gopkg.in/mgo.v2/bson" //在使用与MongoDB有关的函数时，常常对于数据项的查询需要用到关键字“bson”及对应功能
)

/*接下来的几个是和固定资产有关的数据定义，用于数据库管理时数据项相对应*/

/*每一项固定资产的基本数据项，用于储存各数据项的信息，其中具体说明两点：
  1.Row和Cloumn两项在初始化时为0， 只有当csv图导入以后才会及时更新各数据的位置信息
  2.每个类型后面的‘bson’表示各类型数据在MongoDB中的名称
*/
type FixedAsset struct {
	Name             string `bson:"名称"`
	RegistrationDate string `bson:"进账日期"`
	Specification    string `bson:"规格型号"`
	StoredPlace      string `bson:"存放地点"`
	Value            int64  `bson:"资产价值"`
	Row              int    `bson:"Csv图中所在行"`
	Column           int    `bson:"Csv图中所在列"`
}

//这两个是相呼应的类型，其实这个一个二维字符串切片，化成两种类型后，
//由于CSV座位图是一个二位图并且对应的都是字符串信息，
//所以使用两个相关联的结构时在读取CSV文件中的座位表信息时更方便读写和声明变量
type CsvTable struct {
	Records []CsvRow
}
type CsvRow struct {
	Record []string
}

/*main函数是该程序从系统读入的窗口，这个函数主要是有两个功能：
  1.把各功能函数整理好后按顺序放在一起，使得程序运行效率而代码界面整洁
  2.通过标准输出对用户操作进行提示，以免因为读入非目标数据而造成程序崩溃
  3.声明一些变量储存各功能函数运行后返回的值，并且传给其他函数作为参数
*/
func main() {
	var database, collection string //和MongoDB调用相关的变量，database表示数据库名，collection表示集合名
	var n int                       //用于储存用户做出相应选择时的变量
	var randomasset []FixedAsset    //用于储存随机生成的固定资产数据信息
	session, err := mgo.Dial("localhost")
	defer session.Close()
	if err != nil {
		fmt.Println("Failed to open MongoDB application! Program done!")
		return
	}
	fmt.Println("Enter corresponding database and collection you want to open: ")
	fmt.Scanln(&database, &collection)
	c := session.DB(database).C(collection)
	//以上表示MongoDB的启动过程，c用于记录当前所在数据库的所在集合的指针，用于传递给每个功能函数操作
	fmt.Println("\nOpen collection done! Now select a function to maintian the database:")
	fmt.Println("\n1.Manage your opened database\n2.Read CSV seatmap")
	fmt.Println("3.Select some datas randomly according to your percent\n4.Compute the shortest path and print out\nOther.Exit the program\n")
	//以上三句输出提示信息向用户表明程序功能
	for {
		fmt.Println("Enter the number at the head of any discribtion to run corresponding function:")
		fmt.Scanln(&n)
		switch n {
		case 1:
			MangoDBManage(c) //对MongoDB中的数据进行管理
		case 2:
			CsvIntroduce(c) //导入CSV座位图并且修正MongoDB中的各数据项的坐标
		case 3:
			randomasset = RandomSelect(c) //输入抽查比例生成随机数
		case 4:
			GenerateShortestPath(c, randomasset) //最短路径算法和最短路径输出
		default:
			fmt.Println("Program done!")
			return //提示退出程序
		}
	}
	//通过循环使得用户可以多次在程序执行过程中使用不同的功能
}

/*功能函数MongoDBManage是对MongoDB软件的操作函数，主要具有以下功能：
  1.像main函数一样通过输出提示获得用户的输入选择
  2.调用从外部的文本文件直接导入大量固定资产数据信息，方便用户录入
  3.在文本导入的基础上（如果有）查找指定数据并标准输出信息或者输出数据库全部信息
  4.添加额外的数据信息
  5.通过名字修改指定数据项的信息
  6.删除指定数据项或者全部数据项（使用全部删除时会再次进行确认，可以反悔）
*/
func MangoDBManage(c *mgo.Collection) {
	var n int
	chanFixedAsset := IntroduceInformation()    //外部导入文本文件数据，通过通道传送
	judge := SaveInformation(chanFixedAsset, c) //把导入数据存入MongoDB中
	if !judge {
		fmt.Println("Database initializing process failed!")
	}
	fmt.Println("\nEstablishing done! Now select a function to maintian the database:")
	fmt.Println("1.Look up and get some data from the database\n2.Add new information into the database")
	fmt.Println("3.Update the data in the database\n4.Remove some data out of the database\nOther.Exit the manage process")
	fmt.Println("\nEnter the number in front of the describing sentence to select function: ")
	fmt.Scanln(&n)
	switch n {
	case 1:
		FindInformation(c) //查询数据并标准输出显示
	case 2:
		{
			chanFixedAsset1 := AddInformation()
			judge = SaveInformation(chanFixedAsset1, c)
			if !judge {
				fmt.Println("Failed to save information that you added. Managedatabase process done!")
				return
			}
		} //添加数据
	case 3:
		UpdataInformation(c) //修改数据
	case 4:
		RemoveInformation(c) //删除一个或者多个数据
	default:
		fmt.Println("Mangedatabase process done!")
		return
	}
	fmt.Println("Mangedatabase process done!\n")
	return
}

//MongoDBManage函数调用的外部导入函数，用于导入外部文本文件数据
func IntroduceInformation() <-chan FixedAsset {
	var (
		judge, direction string
		input            []byte
		file             *os.File
		err              error
	) //定义导入时需要用的变量
	chanFixedAsset := make(chan FixedAsset, 10)
	fmt.Println("\nIf you want to introduce information into database from a outside text, enter 'yes':")
	fmt.Scanln(&judge) //如果不需要提前导入，可以选择跳过
	if judge != "yes" {
		close(chanFixedAsset)
		return chanFixedAsset
	} else {
		fmt.Println("\nPlease promise your text information placed in the order of 'name, registration date(xxxx.xx.xx), specification, stored place, value'!")
		fmt.Println("Enter the absolute direction of your text:")
		fmt.Scanln(&direction)
		file, err = os.Open(direction) //打开文本文档
		if err != nil {
			fmt.Println("Text is not existed! Introduce process done!")
			close(chanFixedAsset)
			return chanFixedAsset
		} //检查是否错误
		defer file.Close()
		filereader := bufio.NewReader(file)
		for {
			input, _, err = filereader.ReadLine()
			if err == io.EOF {
				close(chanFixedAsset)
				break
			} //读取信息
			s := strings.Split(string(input), " ")    //把一次性读入一行的信息通过space分开，所以要求文档每一行的各个数据用space分开
			value, _ := strconv.ParseInt(s[4], 0, 64) //把价格转换为int型数据
			chanFixedAsset <- FixedAsset{
				Name:             s[0],
				RegistrationDate: s[1],
				Specification:    s[2],
				StoredPlace:      s[3],
				Value:            value}
		}
		fmt.Println("Introdece process done!")
		return chanFixedAsset
	}
}

//MongoDBManage函数调用的添加数据函数，用于在MongoDB中额外添加数据
//注意：从这里添加的数据不提供坐标直接修改，而是需要通过加入CSV座位图后识别修改
func AddInformation() <-chan FixedAsset {
	var name, registrationdate, storedplace, specification, judge string
	var value int64
	chanPerson := make(chan FixedAsset, 10)
	go func() {
		for {
			fmt.Println("\nEnter information in the order of 'name, registration date(xxxx.xx.xx), specification, stored place, value':")
			fmt.Scanln(&name, &registrationdate, &specification, &storedplace, &value)
			chanPerson <- FixedAsset{
				Name:             name,
				RegistrationDate: registrationdate,
				Specification:    specification,
				StoredPlace:      storedplace,
				Value:            value}
			fmt.Println("If wanting to continue, enter 'yes':")
			fmt.Scanln(&judge)
			if judge != "yes" {
				fmt.Println("Enter process done!")
				close(chanPerson)
				break
			}
		}
	}() //这里调用另外一个并发进程的原因是不希望用户输入因思考而将整个程序卡住，用户一边输入时软件方面一边就储存了
	return chanPerson
}

//MongoDBManage函数调用的信息储存函数，用于把导入和新添加的数据存入MongoDB中
func SaveInformation(chanPerson <-chan FixedAsset, c *mgo.Collection) bool {
	for key := range chanPerson {
		err := c.Insert(&key)
		if err != nil {
			fmt.Println("Failed to save information!")
			return false
		}
	}
	return true
}

//MongoDBManage函数调用的查询函数，用于查询指定一项文件或者输出全部文件
func FindInformation(c *mgo.Collection) {
	var result []FixedAsset
	var judge, name string
	var err error
	fmt.Println("Enter 'one' or 'all' to look up a piece of information or all of the information in the database:")
	fmt.Scanln(&judge)  //通过用户的选择来确定输出指定一项或者全部输出
	if judge == "one" { //查询指定一项
		fmt.Println("Enter name of what you want to look for:")
		fmt.Scanln(&name)
		err = c.Find(bson.M{"名称": name}).One(&result[0])
		if err != nil {
			fmt.Println("Fuck off!")
			return
		} else {
			fmt.Println("Name: ", result[0].Name, "\nRegistrationDate: ", result[0].RegistrationDate, "\nSpecification: ", result[0].Specification, "\nStoredPlace: ", result[0].StoredPlace, "\nValue: ", result[0].Value)
			return
		}
	} else { //查询所有数据
		err = c.Find(nil).All(&result)
		if err != nil {
			fmt.Println("Fuck off!")
			return
		} else {
			count := 1
			for _, key := range result {
				fmt.Printf("No.%d\n", count)
				fmt.Println("Name: ", key.Name, "\nRegistrationDate: ", key.RegistrationDate, "\nSpecification: ", key.Specification, "\nStoredPlace: ", key.StoredPlace, "\nValue: ", key.Value)
				count++
			}
			return
		}
	}
}

//MongoDBManage函数调用的修改函数，用于修改MongoDB中已经存在的数据
func UpdataInformation(c *mgo.Collection) {
	var name, kind, information, judge string
	var exist FixedAsset
	var err error
	first := true
	for { //考虑到有可能不止一次修改，使用循环来实现多次
		if !first { //把退出选择放在前面是为了防止放在后面时因为一直找不到数据而无法退出
			fmt.Println("If wanting to continue, enter 'yes':")
			fmt.Scanln(&judge)
			if judge != "yes" {
				fmt.Println("Update process done!")
				return
			}
		} else {
			first = false
		}
		fmt.Println("Enter name of the person you want to updata:")
		fmt.Scanln(&name)
		if err = c.Find(bson.M{"名称": name}).One(&exist); err != nil {
			fmt.Println("Can't find the assumed data from the database! Please try to enter again!")
			continue
		} //如果没找到，重新再来
		fmt.Println("Enter what to updata('name', 'registrationdate', 'specification', 'storedplace', 'value') and the new value: ")
		fmt.Scanln(&kind, &information)
		switch kind {
		case "name":
			err = c.Update(bson.M{"名称": name}, bson.M{"$set": bson.M{"名称": information}})
		case "registrationdate":
			err = c.Update(bson.M{"名称": name}, bson.M{"$set": bson.M{"进账日期": information}})
		case "specification":
			err = c.Update(bson.M{"名称": name}, bson.M{"$set": bson.M{"规格型号": information}})
		case "storedplace":
			err = c.Update(bson.M{"名称": name}, bson.M{"$set": bson.M{"存放地点": information}})
		case "value":
			err = c.Update(bson.M{"名称": name}, bson.M{"$set": bson.M{"资产价值": information}})
		}
		if err != nil {
			fmt.Println("Failed to update assumed data. Update process done!")
			return
		}

	}
}

//MongoDBManage函数调用的删除函数，用于删除指定一项或者所有数据
func RemoveInformation(c *mgo.Collection) {
	var name, judge string
	var err error
	fmt.Println("Enter 'one' or 'all' to look up a piece of information or all of the information in the database:")
	fmt.Scanln(&judge)
	if judge == "one" { //删除指定一项
		fmt.Println("Enter name of what you want to remove:")
		fmt.Scanln(&name)
		err = c.Remove(bson.M{"名称": name})
		if err != nil {
			fmt.Println("Fuck off! You wanna play trick on me?!")
			return
		}
		fmt.Println("Remove process done!")
	} else { //删除所有
		fmt.Println("Are you sure you want to remove ALL the data?")
		fmt.Scanln(&judge)
		if judge != "yes" {
			fmt.Println("Cancel remove operation.")
			return
		} else {
			c.RemoveAll(nil)
			fmt.Println("Remove process done!")
		}
	}
}

/*接下来是第二个模块，CSV座位表的引入。主要包含一个找文件函数和读入文件数据函数，有以下功能：
  1.找到并且读入CSV座位表，识别各数据所在位置
  2.根据识别的位置把MongoDB中存入的对应数据项的横纵坐标值更正
*/
//寻找文件函数
func CsvIntroduce(c *mgo.Collection) {
	var direction string
	var row int
	fmt.Println("Enter the absolute direction of the CSV file:")
	fmt.Scanln(&direction)
	fmt.Println("Enter the total amount of the rows:")
	fmt.Scanln(&row)
	file, err := os.Open(direction)
	if err != nil {
		fmt.Println("File direcction is not exited!")
		return
	} else {
		CsvRead(file, row, c) //调用操作函数
	}
}

//文件读入操作函数实现了把座位表信息表格化，并识别修改个数据的坐标信息
func CsvRead(file *os.File, row int, c *mgo.Collection) {
	csvreader := csv.NewReader(file)      //专门的csv读取器，可以把csv视为一个表格
	allrecord, err := csvreader.ReadAll() //allrecord是一个存有字符串的二维数组
	if err != nil {
		fmt.Println("file reading error!")
		return
	}
	if len(allrecord) < row {
		fmt.Println("The amount of rows is not ample!")
		return
	}
	column := len(allrecord[0])
	records := &CsvTable{make([]CsvRow, row)}
	for i := 0; i < row; i++ {
		record := make([]string, column)
		for j := 0; j < column; j++ {
			record[j] = allrecord[i][j]                                                        //一格一格存入
			_ = c.Update(bson.M{"名称": allrecord[i][j]}, bson.M{"$set": bson.M{"Csv图中所在行": i}}) //更新MongoDB的行数据
			_ = c.Update(bson.M{"名称": allrecord[i][j]}, bson.M{"$set": bson.M{"Csv图中所在列": j}}) //更新MongoDB的列数据
		}
		records.Records[i] = CsvRow{record} //本来records是用来存放在程序中的坐标图，但是目前用不上会报错，所以暂时不管
	}
}

/*接下来是第三个模块，也就是根据比例随机抽取盘查数据。该模块主要有以下功能：
  1.接受用户输入的百分数（用小数表示），并计算总共需要获取的样本数
  2.生成随机数并且以随机数为序号确定随机抽取的数据
*/
//RandomSelect首先从MongoDB中调取所有的数据项作为备选，然后接受用户的比例输入计算抽取个数
//然后调用Random生成一组随机数（各不相同）并以此作为盘查项的序号，从所有备选项中取出，
//存入一个结构组后返回
func RandomSelect(c *mgo.Collection) []FixedAsset {
	var percent float64
	var result []FixedAsset //用于存储从MongoDB调过来的备选项
	fmt.Println("Enter the percent of random-selected sample(please use FLOAT):")
	fmt.Scanln(&percent) //接受比例输入
	err := c.Find(nil).All(&result)
	if err != nil {
		fmt.Println("Can't gain data from database!")
		return nil
	}
	SelectLength := FloatToInt(float64(len(result)) * percent) //计算盘查数量
	RandomArray := make([]int, SelectLength)                   //储存生成的一组随机数
	flag := make([]bool, len(result))
	count := 0
	for {
		i := (Random(count + 79)) % SelectLength
		if !flag[i] {
			RandomArray[count] = i
			count++
			flag[i] = true //初值是false表示没有被选过，一旦选过了就立刻标记为true，从而以后不会再选到
		}
		if count == SelectLength {
			break
		}
	} //设立这个循环的含义是避免因为生成相同的随机数造成选入两个一样的备选项，用flag表示是否被选过就可以避免
	RandomAsset := make([]FixedAsset, SelectLength) //储存随机确定的盘查数据
	fmt.Println("Random fixed asset generated as follows:")
	for i := 0; i < SelectLength; i++ {
		RandomAsset[i] = result[RandomArray[i]] //用随机数来确定选出的数据项
		fmt.Printf("No.%d\n", i+1)
		fmt.Println("Name: ", RandomAsset[i].Name, "\nRegistrationDate: ", RandomAsset[i].RegistrationDate, "\nSpecification: ", RandomAsset[i].Specification, "\nStoredPlace: ", RandomAsset[i].StoredPlace, "\nValue: ", RandomAsset[i].Value)
	}
	fmt.Println("Random selecting process done!\n")
	return RandomAsset
}

//由于用户输入的百分数很有可能导致计算结果不是整数，所以要把计算结果转换为整数
func FloatToInt(f float64) int {
	i, _ := strconv.Atoi(fmt.Sprintf("%1.0f", f))
	return i
}

//生成一组随机数
func Random(extent int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //利用时间变量，使得生成的随机数会不断变化
	return r.Intn(extent)
}

/*接下来是第四个模块，也就是根据确定的盘查数据的坐标位置来计算最小路径并且输出。主要功能有：
  1.从MongoDB中获得所有抽查数据的坐标，并且计算两两坐标后抽象出一个二位数组表示数据所形成的图
  2.根据生成的图利用prim最小生成树算法来确定每一个起点的一条最短路径，由于选取的基本是最短的几条边，所以基本是最短路径
  3.输出每个点为起点的最短路径
*/
//该函数用于从MongoDB中获得所有盘查数据的坐标值，
//计算各点间的直线距离并生成抽象出来的二维数组图中
func GenerateShortestPath(c *mgo.Collection, randomasset []FixedAsset) {
	var csvmap [20][20]int
	var result FixedAsset
	for i := 0; i < len(randomasset); i++ { //根据图的含义我们知道行和列都只表示同一组数据，行表起点列表终点
		var rowi, rowj, columni, columnj int
		for j := 0; j < len(randomasset); j++ {
			c.Find(bson.M{"名称": randomasset[i].Name}).One(&result)
			rowi = result.Row
			columni = result.Column //获取行的横纵坐标
			c.Find(bson.M{"名称": randomasset[j].Name}).One(&result)
			rowj = result.Row
			columnj = result.Column //获取列的横纵坐标
			csvmap[i][j] = (rowi-rowj)*(rowi-rowj) + (columni-columnj)*(columni-columnj)
			//计算距离，由于原函数和其平方单调性相同，这里只需要给出最短路径走法而不用给出具体数值，
			//所以直接采用平方比较大小不用牵扯到float类型数据，简单。
		}
	}
	fmt.Println("The shortest paths are as follows：")
	for i := 0; i < len(randomasset); i++ {
		minpath(i, csvmap, randomasset)
	} //设循环的意思是从每一个起点开始搜寻一边最短路径
	fmt.Println()
}

//寻找最短路径并输出，也就是prim最小生成树算法
func minpath(v int, csvmap [20][20]int, a []FixedAsset) {
	var min, i, j, k int
	lowcost := make([]int, len(a)) //lowcost储存的是从起点到该点的最短路径长度
	closest := make([]int, len(a)) //closest储存的是从起点走最短路径到该点的前一个点，也就是说通过这个数组可以逆向找回起点
	for i = 0; i < len(a); i++ {
		lowcost[i] = csvmap[v][i]
		closest[i] = v
	} //初始化lowcost储存
	first := true
	for i = 1; i < len(a); i++ {
		min = 36767
		for j = 0; j < len(a); j++ {
			if lowcost[j] != 0 && lowcost[j] < min {
				min = lowcost[j]
				k = j
			}
		} //寻找对于区域而言的下一个最短路径点
		if first {
			first = false
			fmt.Printf("%s->%s", a[closest[k]].Name, a[k].Name)
		} else {
			fmt.Printf("->%s", a[k].Name)
		}
		lowcost[k] = 0
		for j = 0; j < len(a); j++ {
			if (lowcost[j] != 0) && (csvmap[k][j] < lowcost[j]) {
				lowcost[j] = csvmap[k][j]
				closest[j] = k
			}
		} //选入最近点以后对于整个区域而言要更改区域对于备选区域的最短路径信息
	}
	fmt.Println()
}
