package seo_test

import (
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/jinzhu/gorm"
	"github.com/qor/admin"
	"github.com/qor/qor"
	"github.com/qor/qor/test/utils"
	"github.com/qor/seo"
)

var db *gorm.DB
var seoCollection *seo.SeoCollection

func init() {
	db = utils.TestDB()
	db.AutoMigrate(&seo.QorSeoSetting{})
}

// Modal
type SeoGlobalSetting struct {
	SiteName  string
	BrandName string
}

type mircoDataInferface interface {
	Render() template.HTML
}

type Category struct {
	Name string
	Seo  seo.Setting `seo:"type:CategoryPage"`
}

// Test Cases
type RenderTestCase struct {
	SiteName   string
	SeoSetting seo.Setting
	Settings   []interface{}
	Result     string
}

type MicroDataTestCase struct {
	MicroDataType string
	ModelObject   interface{}
	HasTag        string
}

// Runner
func TestRender(t *testing.T) {
	setupSeoCollection()
	category := Category{Name: "Clothing", Seo: seo.Setting{Title: "Using Customize Title", EnabledCustomize: false}}
	categoryWithSeo := Category{Name: "Clothing", Seo: seo.Setting{Title: "Using Customize Title", EnabledCustomize: true}}
	var testCases []RenderTestCase
	testCases = append(testCases,
		// Seo setting are empty
		RenderTestCase{"Qor", seo.Setting{Title: "", Description: "", Keywords: ""}, []interface{}{nil, 123}, `<title></title><meta name="description" content=""><meta name="keywords" content=""/>`},
		// Seo setting have value but variables are emptry
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}}", Description: "{{SiteName}}", Keywords: "{{SiteName}}"}, []interface{}{"", ""}, `<title>Qor</title><meta name="description" content="Qor"><meta name="keywords" content="Qor"/>`},
		// Seo setting change Site Name
		RenderTestCase{"ThePlant Qor", seo.Setting{Title: "{{SiteName}}", Description: "{{SiteName}}", Keywords: "{{SiteName}}"}, []interface{}{"", ""}, `<title>ThePlant Qor</title><meta name="description" content="ThePlant Qor"><meta name="keywords" content="ThePlant Qor"/>`},
		// Seo setting have value and variables are present
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}} {{Name}}", Description: "{{URLTitle}}", Keywords: "{{URLTitle}}"}, []interface{}{"Clothing", "/clothing"}, `<title>Qor Clothing</title><meta name="description" content="/clothing"><meta name="keywords" content="/clothing"/>`},
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}} {{Name}} {{Name}}", Description: "{{URLTitle}} {{URLTitle}}", Keywords: "{{URLTitle}} {{URLTitle}}"}, []interface{}{"Clothing", "/clothing"}, `<title>Qor Clothing Clothing</title><meta name="description" content="/clothing /clothing"><meta name="keywords" content="/clothing /clothing"/>`},
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}} {{Name}} {{URLTitle}}", Description: "{{SiteName}} {{Name}} {{URLTitle}}", Keywords: "{{SiteName}} {{Name}} {{URLTitle}}"}, []interface{}{"", ""}, `<title>Qor  </title><meta name="description" content="Qor  "><meta name="keywords" content="Qor  "/>`},
		// Using undefined variables
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}} {{Name1}}", Description: "{{URLTitle1}}", Keywords: "{{URLTitle1}}"}, []interface{}{"Clothing", "/clothing"}, `<title>Qor </title><meta name="description" content=""><meta name="keywords" content=""/>`},
		// Using Resource's seo
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}}", Description: "{{URLTitle}}", Keywords: "{{URLTitle}}"}, []interface{}{category}, `<title>Qor</title><meta name="description" content=""><meta name="keywords" content=""/>`},
		RenderTestCase{"Qor", seo.Setting{Title: "{{SiteName}}", Description: "{{URLTitle}}", Keywords: "{{URLTitle}}"}, []interface{}{categoryWithSeo}, `<title>Using Customize Title</title><meta name="description" content=""><meta name="keywords" content=""/>`},
	)
	i := 1
	for _, testCase := range testCases {
		createGlobalSetting(testCase.SiteName)
		createPageSetting(testCase.SeoSetting)
		metatHTML := string(seoCollection.Render("CategoryPage", testCase.Settings...))
		metatHTML = strings.Replace(metatHTML, "\n", "", -1)
		if string(metatHTML) == testCase.Result {
			color.Green(fmt.Sprintf("Seo Render TestCase #%d: Success\n", i))
		} else {
			t.Errorf(color.RedString(fmt.Sprintf("\nSeo Render TestCase #%d: Failure Result:%s\n", i, string(metatHTML))))
		}
		i += 1
	}
}

func TestMicrodata(t *testing.T) {
	var testCases []MicroDataTestCase
	testCases = append(testCases,
		MicroDataTestCase{"Product", seo.MicroProduct{Name: ""}, `<span itemprop="name"></span>`},
		MicroDataTestCase{"Product", seo.MicroProduct{Name: "Polo"}, `<span itemprop="name">Polo</span>`},
		MicroDataTestCase{"Search", seo.MicroSearch{Target: "http://www.example.com/q={keyword}"}, `http:\/\/www.example.com\/q={keyword}`},
		MicroDataTestCase{"Contact", seo.MicroContact{Telephone: "86-401-302-313"}, `86-401-302-313`},
	)
	i := 1
	for _, microDataTestCase := range testCases {
		tagHTML := reflect.ValueOf(microDataTestCase.ModelObject).Interface().(mircoDataInferface).Render()
		if strings.Contains(string(tagHTML), microDataTestCase.HasTag) {
			color.Green(fmt.Sprintf("Seo Micro TestCase #%d: Success\n", i))
		} else {
			t.Errorf(color.RedString(fmt.Sprintf("\nSeo Micro TestCase #%d: Failure Result:%s\n", i, string(tagHTML))))
		}
		i += 1
	}
}

func setupSeoCollection() {
	seoCollection = seo.New()
	seoCollection.RegisterGlobalSetting(&SeoGlobalSetting{SiteName: "Qor SEO", BrandName: "Qor"})
	seoCollection.RegisterSeo(&seo.Seo{
		Name: "Default Page",
	})
	seoCollection.RegisterSeo(&seo.Seo{
		Name:     "CategoryPage",
		Settings: []string{"Name", "URLTitle"},
		Context: func(objects ...interface{}) (context map[string]string) {
			context = make(map[string]string)
			if len(objects) > 0 && objects[0] != nil {
				if v, ok := objects[0].(string); ok {
					context["Name"] = v
				}
			}
			if len(objects) > 1 && objects[1] != nil {
				if v, ok := objects[1].(string); ok {
					context["URLTitle"] = v
				}
			}
			return context
		},
	})
	Admin := admin.New(&qor.Config{DB: db})
	Admin.AddResource(seoCollection, &admin.Config{Name: "SEO Setting", Menu: []string{"Site Management"}, Singleton: true})
	Admin.MountTo("/admin", http.NewServeMux())
}

func createGlobalSetting(siteName string) {
	globalSeoSetting := seo.QorSeoSetting{}
	db.Where("name = ?", "QorSeoGlobalSettings").Find(&globalSeoSetting)
	globalSetting := make(map[string]string)
	globalSetting["SiteName"] = siteName
	globalSeoSetting.Setting = seo.Setting{GlobalSetting: globalSetting}
	globalSeoSetting.Name = "QorSeoGlobalSettings"
	if db.NewRecord(globalSeoSetting) {
		db.Create(&globalSeoSetting)
	} else {
		db.Save(&globalSeoSetting)
	}
}

func createPageSetting(setting seo.Setting) {
	seoSetting := seo.QorSeoSetting{}
	db.Where("name = ?", "CategoryPage").First(&seoSetting)
	seoSetting.Setting = setting
	if db.NewRecord(seoSetting) {
		db.Create(&seoSetting)
	} else {
		db.Save(&seoSetting)
	}
}
