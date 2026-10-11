package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"learnos/internal/model"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const FinanceMaxBytes = 10 * 1024 * 1024
const FinanceMaxRows = 50000

var FinanceHeaders = []string{"ID", "时间", "分类", "二级分类", "类型", "金额", "币种", "账户1", "账户2", "备注", "已报销", "手续费", "优惠券", "记账者", "账单标记", "标签", "账单图片", "关联账单"}
var ErrFinanceInput = errors.New("invalid financial input")

type FinanceRowError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}
type FinanceParse struct {
	Rows     []model.FinanceTransaction `json:"-"`
	Errors   []FinanceRowError          `json:"errors"`
	Invalid  int                        `json:"invalid"`
	Total    int                        `json:"total"`
	Income   int                        `json:"income"`
	Expense  int                        `json:"expense"`
	Transfer int                        `json:"transfer"`
	From     string                     `json:"from"`
	To       string                     `json:"to"`
	FileHash string                     `json:"-"`
}

var moneyPattern = regexp.MustCompile(`^-?\d{1,10}(\.\d{1,2})?$`)

func FinanceMoney(value string) (int64, error) {
	s := strings.TrimSpace(value)
	if !moneyPattern.MatchString(s) {
		return 0, ErrFinanceInput
	}
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	parts := strings.Split(s, ".")
	whole, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil {
		return 0, ErrFinanceInput
	}
	fraction := int64(0)
	if len(parts) == 2 {
		f := parts[1]
		if len(f) == 1 {
			f += "0"
		}
		fraction, _ = strconv.ParseInt(f, 10, 64)
	}
	v := whole*100 + fraction
	if negative {
		v = -v
	}
	return v, nil
}
func financeDate(s string) bool {
	d, e := time.Parse("2006-01-02", s)
	return e == nil && d.Format("2006-01-02") == s && d.Year() >= 1900 && d.Year() <= 9999
}
func financeTime(s string) (string, error) {
	for _, f := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		d, e := time.Parse(f, s)
		if e == nil && d.Format(f) == s && d.Year() >= 1900 {
			return d.Format("2006-01-02 15:04:05"), nil
		}
	}
	return "", ErrFinanceInput
}
func hashFinance(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func ParseFinanceCSV(data []byte) FinanceParse {
	out := FinanceParse{Rows: []model.FinanceTransaction{}, Errors: []FinanceRowError{}, FileHash: hashFinance(data)}
	badLines := map[int]bool{}
	fail := func(line int, msg string) {
		if !badLines[line] {
			out.Invalid++
			badLines[line] = true
		}
		if len(out.Errors) < 100 {
			out.Errors = append(out.Errors, FinanceRowError{line, msg})
		}
	}
	if len(data) > FinanceMaxBytes {
		fail(1, "文件超过 10 MB")
		return out
	}
	if !utf8.Valid(data) {
		fail(1, "文件必须使用 UTF-8 编码")
		return out
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	header, e := reader.Read()
	if e != nil {
		fail(1, "空文件或表头无法读取")
		return out
	}
	if len(header) != len(FinanceHeaders) {
		fail(1, "表头需要钱迹导出的完整 18 列")
		return out
	}
	for i, h := range header {
		if strings.TrimSpace(h) != FinanceHeaders[i] {
			fail(1, "表头名称或顺序不匹配："+FinanceHeaders[i])
			return out
		}
	}
	seen := map[string]bool{}
	for {
		row, e := reader.Read()
		if e == io.EOF {
			break
		}
		line := 1
		if len(row) > 0 {
			line, _ = reader.FieldPos(0)
		}
		if e != nil {
			var ce *csv.ParseError
			if errors.As(e, &ce) {
				line = ce.Line
			}
			fail(line, "CSV 引号、列数或换行格式错误")
			if len(out.Errors) >= 100 {
				break
			}
			continue
		}
		out.Total++
		if out.Total > FinanceMaxRows {
			fail(line, "最多允许 50000 条交易")
			break
		}
		if len(row) != 18 {
			fail(line, "每条记录必须包含 18 列")
			continue
		}
		bad := false
		check := func(ok bool, msg string) {
			if !ok {
				fail(line, msg)
				bad = true
			}
		}
		id := strings.TrimSpace(row[0])
		check(id != "" && len(id) <= 200, "交易 ID 为空或过长")
		check(!seen[id], "文件内存在重复交易 ID")
		seen[id] = true
		occurred, err := financeTime(strings.TrimSpace(row[1]))
		check(err == nil, "发生时间无效，需为 YYYY-MM-DD HH:mm[:ss]")
		amount, err := FinanceMoney(row[5])
		check(err == nil, "金额无效，最多两位小数，不支持科学计数法")
		for _, i := range []int{11, 12} {
			if strings.TrimSpace(row[i]) != "" {
				_, err := FinanceMoney(row[i])
				check(err == nil, FinanceHeaders[i]+"金额无效")
			}
		}
		kind := map[string]string{"收入": "income", "支出": "expense", "转账": "transfer"}[strings.TrimSpace(row[4])]
		check(kind != "", "类型需为收入、支出或转账")
		currency := strings.TrimSpace(row[6])
		check(regexp.MustCompile(`^[A-Z]{3}$`).MatchString(currency), "币种需为三位大写代码")
		for _, v := range row {
			check(len(v) <= 65536, "单字段超过 64 KB")
			check(!strings.ContainsRune(v, 0), "字段不能包含空字符")
		}
		if bad {
			continue
		}
		tags := []string{}
		tagSeen := map[string]bool{}
		for _, v := range strings.Split(row[15], "#") {
			v = strings.TrimSpace(v)
			if v != "" && !tagSeen[v] {
				tags = append(tags, v)
				tagSeen[v] = true
			}
		}
		raw := map[string]string{}
		for i, h := range FinanceHeaders {
			raw[h] = row[i]
		}
		rawBytes, _ := json.Marshal(raw)
		tagBytes, _ := json.Marshal(tags)
		out.Rows = append(out.Rows, model.FinanceTransaction{ExternalID: id, OccurredAt: occurred, Kind: kind, AmountMinor: amount, Currency: currency, Category: row[2], Subcategory: row[3], Note: row[9], TagsJSON: string(tagBytes), RawJSON: string(rawBytes), ContentHash: hashFinance(rawBytes)})
		switch kind {
		case "income":
			out.Income++
		case "expense":
			out.Expense++
		case "transfer":
			out.Transfer++
		}
		d := occurred[:10]
		if out.From == "" || d < out.From {
			out.From = d
		}
		if d > out.To {
			out.To = d
		}
	}
	if out.Total == 0 {
		fail(2, "文件没有交易记录")
	}
	return out
}
