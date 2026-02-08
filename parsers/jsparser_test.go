package parsers

import (
	"testing"
)

const sampleJSSource = `// A sample JavaScript file
import { useState } from 'react';
import axios from 'axios';

const API_URL = "https://api.example.com";

function hello() {
  console.log("hello");
}

async function fetchData(url) {
  return await fetch(url);
}

export function greet(name) {
  return "Hello, " + name;
}

export default function main() {
  greet("world");
}

export async function loadItems() {
  return [];
}

const handler = (event) => {
  console.log(event);
};

export const processData = async (data) => {
  return data;
};

let mutableState = 0;

class UserService {
  constructor(db) {
    this.db = db;
  }

  async getUser(id) {
    return this.db.find(id);
  }

  deleteUser(id) {
    return this.db.remove(id);
  }

  _internalHelper() {
    return true;
  }
}

export class ApiClient {
  fetch(url) {
    return axios.get(url);
  }
}

function outerFunction() {
  function innerFunction() {
    return true;
  }
  return innerFunction();
}
`

func TestJSParserBasic(t *testing.T) {
	p := &JSParser{}
	symbols, err := p.Parse("sample.js", []byte(sampleJSSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Top-level functions
	assertSymbol(t, byName, "hello", "func", false, "")
	assertSymbol(t, byName, "fetchData", "func", false, "")
	assertSymbol(t, byName, "greet", "func", true, "")
	assertSymbol(t, byName, "main", "func", true, "")
	assertSymbol(t, byName, "loadItems", "func", true, "")

	// Variable declarations
	assertSymbol(t, byName, "API_URL", "const", false, "")
	assertSymbol(t, byName, "handler", "const", false, "")
	assertSymbol(t, byName, "processData", "const", true, "")
	assertSymbol(t, byName, "mutableState", "const", false, "")

	// Classes
	assertSymbol(t, byName, "UserService", "class", false, "")
	assertSymbol(t, byName, "ApiClient", "class", true, "")

	// Methods
	assertSymbol(t, byName, "constructor", "method", true, "UserService")
	assertSymbol(t, byName, "getUser", "method", true, "UserService")
	assertSymbol(t, byName, "deleteUser", "method", true, "UserService")
	assertSymbol(t, byName, "_internalHelper", "method", false, "UserService")
	assertSymbol(t, byName, "fetch", "method", true, "ApiClient")

	// Nested functions should not appear.
	if _, ok := byName["innerFunction"]; ok {
		t.Error("innerFunction should not appear as a top-level symbol")
	}

	// outerFunction should appear
	assertSymbol(t, byName, "outerFunction", "func", false, "")
}

const sampleTSSource = `// TypeScript sample
import type { Request, Response } from 'express';

export interface Config {
  port: number;
  host: string;
  debug?: boolean;
}

interface InternalConfig {
  secret: string;
}

export type Result<T> = Success<T> | Failure;

type InternalID = string;

export enum Status {
  Active = "ACTIVE",
  Inactive = "INACTIVE",
}

const enum Direction {
  Up,
  Down,
  Left,
  Right,
}

export const VERSION = "1.0.0";

export function handleRequest(req: Request, res: Response): void {
  res.send("ok");
}

export abstract class BaseService {
  abstract process(data: unknown): Promise<void>;

  protected log(msg: string) {
    console.log(msg);
  }
}

export class UserController extends BaseService {
  async process(data: unknown): Promise<void> {
    console.log(data);
  }

  public getUsers(): User[] {
    return [];
  }

  private validate(input: string): boolean {
    return input.length > 0;
  }
}

export type Handler = (req: Request) => Response;
`

func TestTSParserBasic(t *testing.T) {
	p := &JSParser{}
	symbols, err := p.Parse("sample.ts", []byte(sampleTSSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Interfaces
	assertSymbol(t, byName, "Config", "interface", true, "")
	assertSymbol(t, byName, "InternalConfig", "interface", false, "")

	// Type aliases
	assertSymbol(t, byName, "Result", "type", true, "")
	assertSymbol(t, byName, "InternalID", "type", false, "")
	assertSymbol(t, byName, "Handler", "type", true, "")

	// Enums
	assertSymbol(t, byName, "Status", "enum", true, "")
	assertSymbol(t, byName, "Direction", "enum", false, "")

	// Constants
	assertSymbol(t, byName, "VERSION", "const", true, "")

	// Functions
	assertSymbol(t, byName, "handleRequest", "func", true, "")

	// Classes
	assertSymbol(t, byName, "BaseService", "class", true, "")
	assertSymbol(t, byName, "UserController", "class", true, "")

	// Methods
	assertSymbol(t, byName, "log", "method", true, "BaseService")
	assertSymbol(t, byName, "process", "method", true, "UserController")
	assertSymbol(t, byName, "getUsers", "method", true, "UserController")
	assertSymbol(t, byName, "validate", "method", false, "UserController")
}

func TestJSParserLineNumbers(t *testing.T) {
	p := &JSParser{}
	symbols, err := p.Parse("sample.js", []byte(sampleJSSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	for _, s := range symbols {
		if s.Line <= 0 {
			t.Errorf("symbol %q has non-positive Line: %d", s.Name, s.Line)
		}
		if s.EndLine < s.Line {
			t.Errorf("symbol %q EndLine (%d) < Line (%d)", s.Name, s.EndLine, s.Line)
		}
	}
}

func TestJSParserEmptyFile(t *testing.T) {
	p := &JSParser{}
	symbols, err := p.Parse("empty.js", []byte(""))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for empty file, got %d", len(symbols))
	}
}

func TestJSParserCommentsOnly(t *testing.T) {
	src := `// This is a comment
// Another comment
/* A block comment */
/*
 * Multi-line
 * block comment
 */
`
	p := &JSParser{}
	symbols, err := p.Parse("comments.js", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(symbols))
	}
}

func TestJSParserExtensions(t *testing.T) {
	p := &JSParser{}
	exts := p.Extensions()
	expected := map[string]bool{".js": true, ".jsx": true, ".ts": true, ".tsx": true}
	if len(exts) != len(expected) {
		t.Errorf("Extensions() = %v, want 4 extensions", exts)
	}
	for _, ext := range exts {
		if !expected[ext] {
			t.Errorf("unexpected extension: %s", ext)
		}
	}
}

func TestJSParserForExtension(t *testing.T) {
	for _, ext := range []string{".js", ".jsx", ".ts", ".tsx"} {
		p := ForExtension(ext)
		if p == nil {
			t.Errorf("ForExtension(%q) returned nil", ext)
		}
	}
}

func TestJSParserReactComponent(t *testing.T) {
	src := `import React from 'react';

export function App() {
  return <div>Hello</div>;
}

export const Header: React.FC = () => {
  return <header>Header</header>;
};

export default function Layout({ children }) {
  return <main>{children}</main>;
}
`
	p := &JSParser{}
	symbols, err := p.Parse("app.tsx", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	assertSymbol(t, byName, "App", "func", true, "")
	assertSymbol(t, byName, "Header", "const", true, "")
	assertSymbol(t, byName, "Layout", "func", true, "")
}

func TestJSParserSignatures(t *testing.T) {
	p := &JSParser{}
	symbols, err := p.Parse("sample.js", []byte(sampleJSSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	tests := []struct {
		name string
		want string
	}{
		{"hello", "function hello() {"},
		{"fetchData", "async function fetchData(url) {"},
		{"greet", "export function greet(name) {"},
		{"UserService", "class UserService"},
		{"ApiClient", "export class ApiClient"},
	}

	for _, tt := range tests {
		sym, ok := byName[tt.name]
		if !ok {
			t.Errorf("symbol %q not found", tt.name)
			continue
		}
		if sym.Signature != tt.want {
			t.Errorf("symbol %q signature = %q, want %q", tt.name, sym.Signature, tt.want)
		}
	}
}

func TestJSParserBracesInStrings(t *testing.T) {
	src := `function test() {
  const x = "{ not a brace }";
  const y = '{ also not }';
  const z = ` + "`template ${expr} literal`" + `;
  return { key: "value" };
}

function afterTest() {
  return true;
}
`
	p := &JSParser{}
	symbols, err := p.Parse("braces.js", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Both functions should be found.
	assertSymbol(t, byName, "test", "func", false, "")
	assertSymbol(t, byName, "afterTest", "func", false, "")
}
