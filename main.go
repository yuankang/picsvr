package main

import (
	"fmt"
	"net/http"
	"os"
	"text/template"
	"path/filepath"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

// 图片信息结构体
type Image struct {
	gorm.Model
	Filename    string
	Description string
}

// 初始化数据库
func initDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", "./pic.db")
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&Image{}) // 自动迁移 创建表
	return db, nil
}

// 上传图片和说明
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// 解析表单
	err := r.ParseMultipartForm(10 << 20) // 允许上传最大10MB的文件
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// 获取 "images" 字段中的多个文件
	files := r.MultipartForm.File["images"] 

	// 获取说明
	description := r.FormValue("description")

	// 创建存储图片的目录
	if _, err := os.Stat("./pic"); os.IsNotExist(err) {
		err := os.MkdirAll("./pic", 0755)
		if err != nil {
			http.Error(w, "Unable to create image directory", http.StatusInternalServerError)
			return
		}
	}

	// 记录图片信息到数据库
	db, err := initDB()
	if err != nil {
		http.Error(w, "Unable to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	for _, fileHeader := range files {
		// 打开上传的文件
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open uploaded file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// 生成文件路径
		fileName := fmt.Sprintf("./pic/%s", filepath.Base(fileHeader.Filename))

		// 创建保存文件
		out, err := os.Create(fileName)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		// 将文件内容写入新文件
		_, err = out.ReadFrom(file)
		if err != nil {
			http.Error(w, "Failed to save file content", http.StatusInternalServerError)
			return
		}

		// 将图片记录到数据库
		image := Image{
			Filename:   fileName,
			Description: description,
		}
		db.Create(&image)
	}

	// 返回上传结果
	fmt.Fprintf(w, "Image uploaded successfully")
}

// 查看所有已上传的图片
func viewImagesHandler(w http.ResponseWriter, r *http.Request) {
    // 获取数据库连接
    db, err := initDB()
    if err != nil {
        http.Error(w, "Unable to connect to the database", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    // 查询所有图片记录
    var images []Image
    db.Find(&images)

    // 读取并渲染 index.html 模板
    tmpl, err := template.ParseFiles("html/index.html")
    if err != nil {
		fmt.Println(err)
        http.Error(w, "Unable to parse template", http.StatusInternalServerError)
        return
    }
	//fmt.Printf("%#v", images)

    // 渲染模板，并传入图片数据
    tmpl.Execute(w, images)
}

// 主函数，设置路由和启动服务器
func main() {
	r := mux.NewRouter()

	// 提供静态文件服务，访问 ./pic/ 目录下的图片文件
	r.PathPrefix("/pic/").Handler(http.StripPrefix("/pic/", http.FileServer(http.Dir("./pic"))))

	// 提供静态文件服务，访问 ./css/ 目录下的 CSS 文件
	r.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))

	// 上传图片和说明的页面
	r.HandleFunc("/upload.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/upload.html")
	}).Methods("GET")

	// 上传图片和说明的处理页面
	r.HandleFunc("/upload", uploadHandler).Methods("POST")

	// 查看已上传图片的页面
	r.HandleFunc("/index.html", viewImagesHandler).Methods("GET")

	// 启动服务器
	http.Handle("/", r)

	port := "8080"
	fmt.Printf("start http server, http://localhost:%s", port)
	http.ListenAndServe(":"+port, nil)
}
