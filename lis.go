package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/peterh/liner"
	"github.com/qjpcpu/fp"
)

/****** main ******/
func main() { repl() }

/****** types ******/
type Expression interface {
	Sexpr() string
}
type Symbol string
type List struct {
	Val  Expression
	Rest *List
}
type Function func(*Env, []Expression) Expression
type UserFunction struct {
	Name string
	Args []string
	Body []Expression
}
type Bool bool
type Integer int64
type Float float64

func (s Symbol) Sexpr() string       { return string(s) }
func (s Function) Sexpr() string     { return `function` }
func (s UserFunction) Sexpr() string { return `user-function:` + s.Name }
func (s Bool) Sexpr() string {
	if s {
		return `true`
	}
	return `false`
}
func (s *List) Sexpr() string {
	buf := "("
	ptr := s
	for ptr.Val != nil {
		if len(buf) > 1 {
			buf += " "
		}
		buf += ptr.Val.Sexpr()
		ptr = ptr.Rest
	}
	return buf + ")"
}
func (s *List) Append(e Expression) {
	if s.Rest == nil {
		s.Rest = &List{}
	}
	if s.Val == nil {
		s.Val = e
		return
	}
	s.Rest.Append(e)
}

func (s Integer) Sexpr() string { return strconv.Itoa(int(s)) }
func (s Float) Sexpr() string   { return fmt.Sprint(s) }

/****** Parsing: parse, tokenize, and read_from_tokens ******/
func parse(program string) Expression {
	return read_from_tokens(tokenize(program))
}

func tokenize(s string) *TokenStream {
	s = strings.ReplaceAll(s, "(", " ( ")
	s = strings.ReplaceAll(s, ")", " ) ")
	return &TokenStream{tokens: fp.StreamOf(strings.Split(s, " ")).Reject(fp.EmptyString()).Strings()}
}

func read_from_tokens(tokens *TokenStream) Expression {
	if tokens.Empty() {
		panic("unexpected EOF while reading")
	}
	token := tokens.Pop()
	if "(" == token {
		l := &List{}
		for tokens.Peek() != ")" {
			l.Append(read_from_tokens(tokens))
		}
		tokens.Pop() // drop ')'
		return l
	} else if ")" == token {
		panic("unexpected )")
	} else {
		return atom(token)
	}
}

func atom(token string) Expression {
	if val, err := strconv.ParseInt(token, 10, 64); err == nil && !strings.Contains(token, ".") {
		return Integer(val)
	} else if fval, err := strconv.ParseFloat(token, 64); err == nil {
		return Float(fval)
	} else if token == `true` || token == `false` {
		return Bool(token == `true`)
	} else {
		return Symbol(token)
	}
}

/****** Environments ******/
type Env struct {
	parent *Env
	scope  map[string]Expression
}

func NewEnv(parent *Env) *Env {
	return &Env{parent: parent, scope: make(map[string]Expression)}
}

func (env *Env) Find(symbol Symbol) map[string]Expression {
	if _, ok := env.scope[string(symbol)]; ok {
		return env.scope
	}
	if env.parent != nil {
		return env.parent.Find(symbol)
	}
	return nil
}

func baseEnv() *Env {
	env := NewEnv(nil)
	env.scope["+"] = Function(plus)
	env.scope["-"] = Function(minus)
	env.scope["*"] = Function(multiple)
	env.scope["/"] = Function(divide)
	env.scope[">"] = Function(gt)
	env.scope["<"] = Function(lt)
	env.scope["=="] = Function(eq)
	return env
}

/****** Interaction: A REPL ******/

func repl() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	env := baseEnv()
	for {
		if sentence, err := line.Prompt("lis.go> "); err == nil {
			val := eval(parse(sentence), env)
			fmt.Println(val.Sexpr())
		} else {
			return
		}
	}
}

/******  eval ******/
func eval(x Expression, env *Env) Expression {
	switch val := x.(type) {
	case Symbol:
		// variable reference
		return env.Find(val)[val.Sexpr()]
	case *List:
		return evalList(val, env)
	default:
		// constant literal
		return x
	}
}

func evalList(x *List, env *Env) Expression {
	name := string(x.Val.(Symbol))
	switch name {
	case "if":
		// (if test conseq alt)
		test, conseq, alt := x.Rest.Val, x.Rest.Rest.Val, x.Rest.Rest.Rest.Val
		if res := eval(test, env); res.(Bool) {
			return eval(conseq, env)
		} else {
			return eval(alt, env)
		}
	case "define":
		// (define var exp)
		vvar, expr := x.Rest.Val, x.Rest.Rest.Val
		env.scope[vvar.Sexpr()] = eval(expr, env)
		return env.scope[vvar.Sexpr()]
	case "set!":
		// (set! var exp)
		vvar, expr := x.Rest.Val, x.Rest.Rest.Val
		env.Find(vvar.(Symbol))[vvar.Sexpr()] = expr
		return expr
	case "define-func":
		var userf UserFunction
		// (define-func name (arg1 arg2) body)
		name := string(x.Rest.Val.(Symbol))
		var args []string
		argsExpr := x.Rest.Rest.Val.(*List)
		for argsExpr.Val != nil {
			args = append(args, string(argsExpr.Val.(Symbol)))
			argsExpr = argsExpr.Rest
		}
		userf.Name = name
		userf.Args = args
		expr := x.Rest.Rest.Rest
		for expr.Val != nil {
			userf.Body = append(userf.Body, expr.Val)
			expr = expr.Rest
		}
		env.scope[name] = userf
		return userf
	default:
		// (function arg...)
		proc := eval(x.Val, env)
		var args []Expression
		for ptr := x.Rest; ptr.Val != nil; {
			args = append(args, eval(ptr.Val, env))
			ptr = ptr.Rest
		}
		if uf, ok := proc.(UserFunction); ok {
			return callUserFunction(env, uf, args)
		}
		return proc.(Function)(NewEnv(env), args)
	}
}

func callUserFunction(env *Env, f UserFunction, args []Expression) Expression {
	env = NewEnv(env)
	for i, arg := range f.Args {
		env.scope[arg] = args[i]
	}
	var ret Expression
	for _, expr := range f.Body {
		ret = eval(expr, env)
	}
	return ret
}

type TokenStream struct {
	tokens []string
	idx    int
}

func (t *TokenStream) Pop() string {
	t.idx++
	return t.tokens[t.idx-1]
}

func (t *TokenStream) Empty() bool {
	return t.idx == len(t.tokens)
}

func (t *TokenStream) Peek() string {
	return t.tokens[t.idx]
}

/***** functions ***********/
func plus(env *Env, exprs []Expression) Expression {
	if isFloat(exprs[0]) || isFloat(exprs[1]) {
		return Float(toFloat(exprs[0]) + toFloat(exprs[1]))
	}
	return Integer(toInt(exprs[0]) + toInt(exprs[1]))
}

func minus(env *Env, exprs []Expression) Expression {
	if isFloat(exprs[0]) || isFloat(exprs[1]) {
		return Float(toFloat(exprs[0]) - toFloat(exprs[1]))
	}
	return Integer(toInt(exprs[0]) - toInt(exprs[1]))
}

func multiple(env *Env, exprs []Expression) Expression {
	if isFloat(exprs[0]) || isFloat(exprs[1]) {
		return Float(toFloat(exprs[0]) * toFloat(exprs[1]))
	}
	return Integer(toInt(exprs[0]) * toInt(exprs[1]))
}

func divide(env *Env, exprs []Expression) Expression {
	if isFloat(exprs[0]) || isFloat(exprs[1]) {
		return Float(toFloat(exprs[0]) / toFloat(exprs[1]))
	}
	return Integer(toInt(exprs[0]) / toInt(exprs[1]))
}

func gt(env *Env, exprs []Expression) Expression {
	return Bool(toFloat(exprs[0]) > toFloat(exprs[1]))
}

func lt(env *Env, exprs []Expression) Expression {
	return Bool(toFloat(exprs[0]) < toFloat(exprs[1]))
}
func eq(env *Env, exprs []Expression) Expression {
	return Bool(toFloat(exprs[0]) == toFloat(exprs[1]))
}
func isFloat(e Expression) bool {
	_, ok := e.(Float)
	return ok
}

func toFloat(e Expression) float64 {
	if isFloat(e) {
		return float64(e.(Float))
	}
	return float64(e.(Integer))
}

func toInt(e Expression) int64 {
	if isFloat(e) {
		return int64(e.(Float))
	}
	return int64(e.(Integer))
}
