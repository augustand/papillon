package handler

import (
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gogank/papillon/configuration"
	"github.com/gogank/papillon/publish"
	"github.com/gogank/papillon/render"
	"github.com/gogank/papillon/utils"
	"github.com/mrunalp/fileutils"
)

//Generate generate the whole source path
func Generate(confPath string) error {
	cnf := config.NewConfig(confPath)

	sourceDir := cnf.GetString(utils.DirSource)
	postsDir := cnf.GetString(utils.DirPosts)
	publicDir := cnf.GetString(utils.DirPublic)
	themeDir := cnf.GetString(utils.DirTheme)

	// 1. 检查 source 文件夹是否存在
	if !utils.ExistDir(sourceDir) {
		return fmt.Errorf("source directory '%s' doesn't exist, cann't generate", sourceDir)
	}

	// 2. 删除 public 文件夹
	if utils.ExistDir(publicDir) {
		if err := utils.RemoveDir(publicDir); err != nil {
			return err
		}
	}

	// 3. 创建新的 public 文件夹
	if !utils.Mkdir(publicDir) {
		return fmt.Errorf("create directory %s failed", publicDir)
	}

	if utils.ExistDir(postsDir) {

		// 4. 创建 public/posts 文件夹
		if !utils.Mkdir(path.Join(publicDir, "posts")) {
			return fmt.Errorf("create directory %s failed", path.Join(publicDir, "posts"))
		}

		// 5. 遍历source/posts/ 目录中的所有的markdown文件， 转化为html文件
		files, err := utils.ListDir(postsDir, "md")
		if err != nil {
			return err
		}

		parse := render.New()

		// 6. 复制样式文件
		if err := fileutils.CopyDirectory(path.Join(themeDir, "assets"), publicDir); err != nil {
			return err
		}

		// 生成文章的静态html
		for _, fname := range files {
			mdContent, err := utils.ReadFile(path.Join(postsDir, fname))
			if err != nil {
				return err
			}

			//postsTpl, err := utils.ReadFile(path.Join(themeDir, "post.hbs"))
			postsTpl, err := utils.ReadFile(path.Join(themeDir, "post2.hbs"))
			if err != nil {
				return err
			}

			// 调用markdown－>html方法, 得到文章信息、文章内容
			//fileInfo, htmlContent, err := parse.DoRender(mdContent, postsTpl, nil)
			postCtx := make(map[string]interface{})
			postCtx["blogTitle"] = cnf.GetString(utils.CommonTitle)
			postCtx["blogDesc"] = cnf.GetString(utils.CommonDesc)
			postCtx["blogAuthor"] = cnf.GetString(utils.CommonAuthor)
			postCtx["articlesCount"] = strconv.Itoa(len(files))

			fileInfo, htmlContent, err := parse.DoRender(mdContent, postsTpl, postCtx)
			if err != nil {
				return err
			}

			now := time.Now()
			year := strconv.Itoa(now.Year())
			month := strconv.Itoa(int(now.Month()))
			day := strconv.Itoa(now.Day())
			title := "Untitled" + strconv.Itoa(rand.Int())

			// 根据文章信息创建文件夹
			for k, v := range fileInfo {

				// 确定日期文件夹目录
				if k == "date" {
					ds := strings.Split(v.(string), "/")

					if len(ds) == 3 {
						year = ds[0]
						month = ds[1]
						day = ds[2]
					}
				}

				// 确定文章文件夹目录
				if k == "title" {
					title = v.(string)
				}
			}

			// 检查年份文件夹是否存在
			if !utils.ExistDir(path.Join(publicDir, "posts", year)) {
				if !utils.Mkdir(path.Join(publicDir, "posts", year)) {
					return fmt.Errorf("create directory %s failed", path.Join(publicDir, "posts", year))
				}
			}

			// 检查月份文件夹是否存在
			if !utils.ExistDir(path.Join(publicDir, "posts", year, month)) {
				if !utils.Mkdir(path.Join(publicDir, "posts", year, month)) {
					return fmt.Errorf("create directory %s failed", path.Join(publicDir, "posts", year, month))
				}
			}

			// 检查日期文件夹是否存在
			if !utils.ExistDir(path.Join(publicDir, "posts", year, month, day)) {
				if !utils.Mkdir(path.Join(publicDir, "posts", year, month, day)) {
					return fmt.Errorf("create directory %s failed", path.Join(publicDir, "posts", year, month, day))
				}
			}

			newTitle := strings.Replace(title, " ", "_", -1)
			if !utils.Mkdir(path.Join(publicDir, "posts", year, month, day, newTitle)) {
				return fmt.Errorf("create directory %s failed",
					path.Join(publicDir, "posts", year, month, day, newTitle))
			}

			// 根据文章内容创建html文件
			newHTMLContent, err := parse.ConvertLink(htmlContent, publicDir)
			if err != nil {
				return err
			}

			if !utils.Mkfile(path.Join(publicDir, "posts", year, month, day, newTitle, "index.html"), newHTMLContent) {
				return fmt.Errorf("create file %s failed",
					path.Join(publicDir, "posts", year, month, day, newTitle, "index.html"))
			}
		}

		// 7. 生成首页的html
		if err := genIndexHTML(cnf, path.Join(publicDir, "index.html")); err != nil {
			return err
		}
	}
	return nil
}

func genIndexHTML(cnf *config.Config, indexPath string) error {
	parse := render.New()

	themeDir := cnf.GetString(utils.DirTheme)
	postsDir := cnf.GetString(utils.DirPosts)
	publicDir := cnf.GetString(utils.DirPublic)

	indexCtx := make(map[string]interface{})

	// 首页的基本信息
	indexCtx["title"] = cnf.GetString(utils.CommonTitle)
	indexCtx["description"] = cnf.GetString(utils.CommonDesc)
	indexCtx["author"] = cnf.GetString(utils.CommonAuthor)

	// 首页的文章信息
	files, err := utils.ListDir(postsDir, "md")
	if err != nil {
		return fmt.Errorf("read directory %s failed", postsDir)
	}

	indexCtx["articles"] = make([]map[string]interface{}, len(files))
	indexCtx["articlesCount"] = len(files)

	for i, fname := range files {
		mdContent, err := utils.ReadFile(path.Join(postsDir, fname))
		if err != nil {
			return err
		}

		var dateSlice []string
		if meta, err := render.GetMeta(mdContent); err == nil {
			date := meta["date"]
			title := strings.Replace(meta["title"], " ", "_", -1)
			//title := meta["title"]

			dateSlice = strings.Split(date, "/")

			indexCtx["articles"].([]map[string]interface{})[i] = make(map[string]interface{})
			indexCtx["articles"].([]map[string]interface{})[i]["date"] = date
			indexCtx["articles"].([]map[string]interface{})[i]["title"] = meta["title"]
			indexCtx["articles"].([]map[string]interface{})[i]["abstract"] = meta["abstract"]

			articleURL := path.Join("posts", dateSlice[0], dateSlice[1], dateSlice[2], title, "index.html")
			indexCtx["articles"].([]map[string]interface{})[i]["url"] = "/" + articleURL
		} else {
			return err
		}
	}

	indexTpl, err := utils.ReadFile(path.Join(themeDir, "index.hbs"))
	if err != nil {
		return err
	}

	_, indexHTML, err := parse.DoRender(nil, indexTpl, indexCtx)
	if err != nil {
		return err
	}

	newIndexHTML, err := parse.ConvertLink(indexHTML, publicDir)
	if err != nil {
		return err
	}

	if !utils.Mkfile(indexPath, newIndexHTML) {
		return fmt.Errorf("create file %s failed", indexPath)
	}
	pub := publish.NewImpl()
	indexHash, err := pub.AddFile(indexPath)

	if err != nil {
		return err
	}

	fmt.Println("convert index.html to https://ipfs.io/ipfs/" + indexHash)

	return nil
}
