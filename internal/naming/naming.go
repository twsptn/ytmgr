package naming

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Info struct {
	Date  time.Time
	Title string
	Smin  int
	Ssec  int
	Emin  int
	Esec  int
}

func ProperName(name string, ext string) string {
	if !strings.ContainsAny(name, "()[]") {
		rs := []rune(name)
		first := ""
		for i, v := range rs {
			if len(first) == 0 && i > 8 && unicode.IsDigit(v) {
				first = string(rs[:i])
				fmt.Println("found digit:", i, "-", string(rs[:i]))
			} else if len(first) > 0 && v == '.' {
				j := utf8.RuneCountInString(first)

				name = first + "(" + string(rs[j:i]) + ")" + ext
				//  fmt.Println(j, " > ", i, first+"("+string(rs[j:i])+")"+ext)
				break
			}
		}

	}
	info := ExtractName2(name)
	var bldr strings.Builder
	dateStr := info.Date.Format("zh060102")
	bldr.WriteString(dateStr)
	tstr := fmt.Sprintf("_[%02d.%02d-%02d.%02d]_", info.Smin, info.Ssec, info.Emin, info.Esec)
	bldr.WriteString(tstr)
	bldr.WriteString(info.Title)
	bldr.WriteString(ext)
	return bldr.String()

}

func ArchiveMonth(tm time.Time) string {
	layout := "2006_01"
	return tm.Format(layout)
}

// 當父母生病時我們要如何做？- zh220731（07_40--14_20）)
// 我們捫心自問修行是為了離苦還是快樂 - zh220813( 00_00--04_07)
func ExtractName2(str string) Info {
	str = strings.ReplaceAll(str, "（", "(")
	str = strings.ReplaceAll(str, "）", ")")
	ret := Info{}
	dateStr, titleStr, timeStr := "", "", ""

	matches, err := extract2(str, `^(zh\d\d\d\d\d\d)(.*?)\((.*?)\)`)
	if err == nil {
		dateStr, titleStr, timeStr = matches[1], matches[2], matches[3]
		goto RETURN
	}

	matches, err = extract2(str, `^(zh\d\d\d\d\d\d)_\[(.*?)\]_([^\.]*)`)
	if err == nil {
		dateStr, timeStr, titleStr = matches[1], matches[2], matches[3]
		goto RETURN
	}

	matches, err = extract2(str, `^(zh\d\d\d\d\.\d\d\.\d\d)(.*?)\((.*?)\)`)
	if err == nil {
		dateStr, titleStr, timeStr = matches[1], matches[2], matches[3]
		goto RETURN
	}

	matches, err = extract2(str, `(.*?) *- *(zh\d\d\d\d\d\d)\((.*?)\)`)
	if err == nil {
		dateStr, titleStr, timeStr = matches[1], matches[2], matches[3]
		goto RETURN
	}

	log.Panic(errors.New("unknown file name format: " + str))

RETURN:
	// fmt.Println("name is: ", str)
	// fmt.Println("matches is: ", matches)
	ret.Date = extractDate(dateStr)
	ret.Title = titleStr
	ret.Smin, ret.Ssec, ret.Emin, ret.Esec = extractTime(timeStr)
	return ret
}

func extractDate(str string) time.Time {
	tm, err := time.Parse(layout, str)
	if err != nil {
		tm, err = time.Parse(layout2, str)
		if err != nil {
			log.Panic(err)
		}
	}
	return tm
}

func extractTime(str string) (int, int, int, int) {
	matches, err := extract2(str, `\D*(\d+)(?:\D)+(\d+)(?:\D+)(\d+)(?:\D+)(\d+)(?:\D+)(\d+)(?:\D+)(\d+)`)
	if err == nil {
		return Atoi(matches[1])*60 + Atoi(matches[2]), Atoi(matches[3]), Atoi(matches[4])*60 + Atoi(matches[5]), Atoi(matches[6])
	}

	matches, err = extract2(str, `\D*(\d+)(?:\D)+(\d+)(?:\D+)(\d+)(?:\D+)(\d+)(?:\D+)(\d+)`)
	if err == nil {
		return Atoi(matches[1]), Atoi(matches[2]), Atoi(matches[3])*60 + Atoi(matches[4]), Atoi(matches[5])
	}

	matches, err = extract2(str, `\D*(\d+)(?:\D)+(\d+)(?:\D+)(\d+)(?:\D+)(\d+)`)
	if err == nil {
		return Atoi(matches[1]), Atoi(matches[2]), Atoi(matches[3]), Atoi(matches[4])
	}

	log.Panic(errors.New("can't match video time"), str)

	return 0, 0, 0, 0
}

const layout = "zh060102"
const layout2 = "zh2006.01.02"

func Atoi(str string) int {
	ret, err := strconv.Atoi(str)
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func extract2(str string, regex string) ([]string, error) {
	re := regexp.MustCompile(regex)
	res := re.FindAllStringSubmatch(str, -1)
	if len(res) != 1 {
		return []string{}, errors.New("bad")
	}
	return res[0], nil

}
