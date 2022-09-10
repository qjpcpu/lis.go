How to Write a (Lisp) Interpreter (in Golang)
==================================================

# Try REPL

```
go build && ./lis-go
```

# Atoms

This tiny interpreter support 4 atoms:
* symbol
* boolean
* integer and float

# Operations

This tiny interpreter support 10 operations:

* `if`

``` clojure
(if condition true-expr false-expr)

(if (> 2 1) 2 1) ; return 2
```

* `define`

define a variable.

``` clojure
(define var expr)

(define number 12)  ; now number bind to 12
```


* `set!`

update an exist variable.

``` clojure
(set! var expr)

(set! number 13) ; now number is 13
```


* `+` `-` `*` `/`

arithmetic operations add/minus/multiply/divide.

* `>` `<` `==`

compare operations greater than/less than/equal.


# In production

This is just a simple demo for study. If you want an interpreter in production,
please use [glisp](https://github.com/qjpcpu/glisp), and refer to its documentation [glisp wiki](https://github.com/qjpcpu/glisp/wiki).
