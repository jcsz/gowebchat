package mysql

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MYSQL_SELECT_FIELD   = "field"
	MYSQL_SELECT_CONDS   = "conds"
	MYSQL_SELECT_LIMIT   = "limit"
	MYSQL_SELECT_OFFSET  = "offset"
	MYSQL_SELECT_ORDERBY = "orderby"
	MYSQL_SELECT_GROUPBY = "groupby"
	MYSQL_SELECT_HAVING  = "having"
)

type OrderMap struct {
	keys []string
	_map map[string]interface{}
}

func NewOrderMap() *OrderMap {
	om := &OrderMap{
		_map: make(map[string]interface{}),
	}
	return om
}

func (om *OrderMap) Set(key string, value interface{}) *OrderMap {
	if _, ok := om._map[key]; !ok {
		om.keys = append(om.keys, key)
	}
	om._map[key] = value
	return om
}

func (om *OrderMap) Keys() []string {
	return om.keys
}

func (om *OrderMap) Values() []interface{} {
	var value []interface{}
	for _, v := range om.keys {
		value = append(value, om._map[v])
	}

	return value
}
func (om *OrderMap) Maps() map[string]interface{} {
	return om._map
}

/* interface{} */
type _sqlExpr interface {
	Set(key string, value interface{})
	Prepare() (string, error)
	ExecVal() []interface{}
}

/*condition*/
type Condition struct {
	val *OrderMap
}

func NewCondition() *Condition {
	cond := &Condition{
		val: NewOrderMap(),
	}
	return cond
}

func (c *Condition) Set(cond string, value interface{}) {
	c.val.Set(fmt.Sprintf("%s ?", cond), value)
}

func (c *Condition) SetIn(cond string, value []string) {
	var s []string
	var omap *OrderMap = NewOrderMap()
	for k, v := range value {
		s = append(s, "?")
		omap.Set(fmt.Sprintf("IN:%s", k), v)
	}
	c.val.Set(fmt.Sprintf("%s IN (%s)", cond, strings.Join(s, ",")), *omap)
}

func (c *Condition) SetNotIn(cond string, value []string) {
	var s []string
	var omap *OrderMap = NewOrderMap()
	for k, v := range value {
		s = append(s, "?")
		omap.Set(fmt.Sprintf("NOTIN:%s", k), v)
	}
	c.val.Set(fmt.Sprintf("%s NOT IN (%s)", cond, strings.Join(s, ",")), *omap)
}

func (c *Condition) Prepare() (string, error) {
	keys := c.val.Keys()
	if len(keys) == 0 {
		return "", nil
	}

	sql := fmt.Sprintf("WHERE %s", strings.Join(keys, " and "))

	return sql, nil
}
func (c *Condition) ExecVal() []interface{} {
	var r []interface{}
	for _, v := range c.val.Values() {
		t := fmt.Sprintf("%T", v)
		if t == "mysql.OrderMap" {
			omap := v.(OrderMap)
			p := &omap
			for _, vv := range p.Values() {
				r = append(r, vv)
			}
		} else {
			r = append(r, v)
		}
	}
	return r
}

/* field */
type Field struct {
	val  *OrderMap
	isup bool
}

func NewField() *Field {
	f := &Field{
		val:  NewOrderMap(),
		isup: false,
	}
	return f
}
func (f *Field) Set(cond string, value interface{}) {
	f.val.Set(cond, value)
}
func (f *Field) SetUp() {
	f.isup = true
}
func (f *Field) Prepare() (string, error) {
	keys := f.val.Keys()
	if len(keys) == 0 {
		return "*", nil
	}
	sql := fmt.Sprintf("%s", strings.Join(keys, ","))
	return sql, nil
}
func (f *Field) PrepareSet() (string, error) {
	keys := f.val.Keys()
	if len(keys) == 0 {
		return "", fmt.Errorf("have no set value")
	}
	var sets []string
	for _, v := range keys {
		sets = append(sets, fmt.Sprintf("%s = ?", v))
	}

	sql := fmt.Sprintf("SET %s ", strings.Join(sets, ","))

	return sql, nil
}

func (f *Field) ExecVal() []interface{} {
	var r []interface{}
	if f.isup {
		return f.val.Values()
	}
	return r
}

/* limit offse*/
type LimitOffset struct {
	val *OrderMap
}

func NewLimitOffset() *LimitOffset {
	l := &LimitOffset{
		val: NewOrderMap(),
	}
	return l
}
func (l *LimitOffset) Set(cond string, value interface{}) {
	l.val.Set(cond, value)
}
func (l *LimitOffset) Prepare() (string, error) {
	maps := l.val.Maps()
	if len(maps) == 0 {
		return "", nil
	}
	_, ok_offset := maps[MYSQL_SELECT_OFFSET]
	_, ok_limit := maps[MYSQL_SELECT_LIMIT]

	if ok_offset && !ok_limit {
		return "", fmt.Errorf("offset is set but limit not set")
	}

	if ok_offset {
		return "LIMIT ?,? ", nil
	}

	return "LIMIT ? ", nil
}
func (l *LimitOffset) ExecVal() []interface{} {
	var r []interface{}
	maps := l.val.Maps()
	if len(maps) == 0 {
		return r
	}
	v_offset, ok_offset := maps[MYSQL_SELECT_OFFSET]
	v_limit, ok_limit := maps[MYSQL_SELECT_LIMIT]
	if ok_limit && ok_offset {
		r = append(r, v_offset, v_limit)
	} else if ok_limit && !ok_offset {
		r = append(r, v_limit)
	}

	return r
}

/* groupby having */
type GroupByHaving struct {
	groupby string
	val     *OrderMap
}

func NewGroupByHaving() *GroupByHaving {
	g := &GroupByHaving{
		groupby: "",
		val:     NewOrderMap(),
	}
	return g
}
func (g *GroupByHaving) Set(cond string, value interface{}) {
	if cond == MYSQL_SELECT_GROUPBY {
		g.groupby = value.(string)
	} else {
		g.val.Set(cond, value)
	}
}
func (g *GroupByHaving) Prepare() (string, error) {
	if g.groupby == "" {
		return "", nil
	}
	keys := g.val.Keys()
	var having []string
	for _, v := range keys {
		having = append(having, fmt.Sprintf("%s ?", v))
	}

	if len(keys) > 0 {
		return fmt.Sprintf("GROUP BY %s HAVING %s ", g.groupby, strings.Join(having, " and ")), nil
	}

	return fmt.Sprintf("GROUP BY %s ", g.groupby), nil
}
func (g *GroupByHaving) ExecVal() []interface{} {
	var r []interface{}
	if g.groupby == "" {
		return r
	}
	return g.val.Values()
}

/* Select map */
type SqlExpr struct {
	Field         *Field
	Conds         *Condition
	LimitOffset   _sqlExpr
	GroupbyHaving _sqlExpr
	Orderby       string
}

func NewSqlExpr() *SqlExpr {
	se := &SqlExpr{
		Field:         NewField(),
		Conds:         NewCondition(),
		LimitOffset:   NewLimitOffset(),
		GroupbyHaving: NewGroupByHaving(),
	}
	return se
}
func (se *SqlExpr) SetField(fields ...string) *SqlExpr {
	for _, v := range fields {
		se.Field.Set(v, v)
	}
	return se
}
func (se *SqlExpr) SetFieldUp(f, v string) *SqlExpr {
	se.Field.Set(f, v)
	return se
}
func (se *SqlExpr) SetCondition(cond string, value interface{}) *SqlExpr {
	se.Conds.Set(cond, value)
	return se
}
func (se *SqlExpr) SetCondIn(f string, v []string) *SqlExpr {
	se.Conds.SetIn(f, v)
	return se
}
func (se *SqlExpr) SetCondNotIn(f string, v []string) *SqlExpr {
	se.Conds.SetNotIn(f, v)
	return se
}
func (se *SqlExpr) SetLimit(limit int64) *SqlExpr {
	se.LimitOffset.Set(MYSQL_SELECT_LIMIT, limit)
	return se
}
func (se *SqlExpr) SetOffset(offset int64) *SqlExpr {
	se.LimitOffset.Set(MYSQL_SELECT_OFFSET, offset)
	return se
}
func (se *SqlExpr) SetOrderBy(orderby string) *SqlExpr {
	se.Orderby = orderby
	return se
}
func (se *SqlExpr) SetGroupBy(groupby string) *SqlExpr {
	se.GroupbyHaving.Set(MYSQL_SELECT_GROUPBY, groupby)
	return se
}
func (se *SqlExpr) SetHaving(cond string, value interface{}) *SqlExpr {
	se.GroupbyHaving.Set(cond, value)
	return se
}

func (se *SqlExpr) GetPrepareSql(table string) (string, error) {
	var (
		err error
		s   string
	)
	s, err = se.Field.Prepare()
	if err != nil {
		return "", err
	}
	s_field := s

	s, err = se.Conds.Prepare()
	if err != nil {
		return "", err
	}
	s_cond := s

	s, err = se.GroupbyHaving.Prepare()
	if err != nil {
		return "", err
	}
	s_grouphaving := s

	s, err = se.LimitOffset.Prepare()
	if err != nil {
		return "", err
	}
	s_limitoffset := s

	s_orderby := ""
	if se.Orderby != "" {
		s_orderby = fmt.Sprintf("ORDER BY %s", se.Orderby)
	}

	re, _ := regexp.Compile(`\s{2,}`)
	sql := fmt.Sprintf("SELECT %s FROM %s %s %s %s %s", s_field, table, s_cond, s_grouphaving, s_orderby, s_limitoffset)
	sql = re.ReplaceAllString(sql, " ")
	sql = strings.TrimRight(sql, " ")
	return sql, nil
}

func (se *SqlExpr) GetUpdateSql(table string) (string, error) {
	var (
		err error
		s   string
	)
	s, err = se.Field.PrepareSet()
	if err != nil {
		return "", err
	}
	s_field := s

	s, err = se.Conds.Prepare()
	if err != nil {
		return "", err
	}
	s_cond := s

	s, err = se.GroupbyHaving.Prepare()
	if err != nil {
		return "", err
	}
	s_grouphaving := s

	s, err = se.LimitOffset.Prepare()
	if err != nil {
		return "", err
	}
	s_limitoffset := s

	s_orderby := ""
	if se.Orderby != "" {
		s_orderby = fmt.Sprintf("ORDER BY %s", se.Orderby)
	}

	re, _ := regexp.Compile(`\s{2,}`)
	sql := fmt.Sprintf("UPDATE %s %s %s %s %s %s", table, s_field, s_cond, s_grouphaving, s_orderby, s_limitoffset)
	sql = re.ReplaceAllString(sql, " ")
	sql = strings.TrimRight(sql, " ")
	return sql, nil
}

func (se *SqlExpr) GetDeleteSql(table string) (string, error) {
	var (
		err error
		s   string
	)

	s, err = se.Conds.Prepare()
	if err != nil {
		return "", err
	}
	s_cond := s

	s, err = se.GroupbyHaving.Prepare()
	if err != nil {
		return "", err
	}
	s_grouphaving := s

	s, err = se.LimitOffset.Prepare()
	if err != nil {
		return "", err
	}
	s_limitoffset := s

	s_orderby := ""
	if se.Orderby != "" {
		s_orderby = fmt.Sprintf("ORDER BY %s", se.Orderby)
	}

	re, _ := regexp.Compile(`\s{2,}`)
	sql := fmt.Sprintf("DELETE FROM  %s %s %s %s %s", table, s_cond, s_grouphaving, s_orderby, s_limitoffset)
	sql = re.ReplaceAllString(sql, " ")
	sql = strings.TrimRight(sql, " ")
	return sql, nil
}

func (se *SqlExpr) ExecVal() []interface{} {
	var ret []interface{}
	for _, v := range se.Field.ExecVal() {
		ret = append(ret, v)
	}
	for _, v := range se.Conds.ExecVal() {
		ret = append(ret, v)
	}
	for _, v := range se.GroupbyHaving.ExecVal() {
		ret = append(ret, v)
	}
	for _, v := range se.LimitOffset.ExecVal() {
		ret = append(ret, v)
	}
	return ret
}
