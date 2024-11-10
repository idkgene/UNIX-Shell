package shell

import (
	"fmt"
	"os"
	"strings"
)

type Parser struct {
    shell *Shell
}

type Token struct {
    Type  TokenType
    Value string
}

type TokenType int

const (
	TokenWord TokenType = iota
	TokenPipe
	TokenRedirectIn
	TokenRedirectOut
	TokenRedirectAppend
	TokenBackground
	TokenAnd
	TokenOr
	TokenSemicolon
	TokenQuote
	TokenVariable
)

type ParseError struct {
	Message string
	Pos     int
}

type Command struct {
	Args         []string
	Stdin        string
	Stdout       string
	StdoutAppend bool
	Background   bool
	Env          []string
	Dir          string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Pos, e.Message)
}

func NewParser(shell *Shell) *Parser {
	return &Parser{shell: shell}
}

func (p *Parser) Parse(input string) ([]Command, error) {
	tokens, err := p.tokenize(input)
	if err != nil {
			return nil, err
	}

	return p.parseTokens(tokens)
}

func (p *Parser) tokenize(input string) ([]Token, error) {
	var tokens []Token
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for pos, char := range input {
			switch {
			case char == '"' || char == '\'':
					if !inQuote {
							inQuote = true
							quoteChar = char
					} else if char == quoteChar {
							inQuote = false
							tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
							current.Reset()
					} else {
							current.WriteRune(char)
					}

			case char == '|' && !inQuote:
					if current.Len() > 0 {
							tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
							current.Reset()
					}
					
					tokens = append(tokens, Token{Type: TokenPipe, Value: "|"})

			case char == '>' && !inQuote:
					if pos+1 < len(input) && input[pos+1] == '>' {
							if current.Len() > 0 {
									tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
									current.Reset()
							}
							tokens = append(tokens, Token{Type: TokenRedirectAppend, Value: ">>"})
							pos++ // Skip next character
					} else {
							if current.Len() > 0 {
									tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
									current.Reset()
							}
							tokens = append(tokens, Token{Type: TokenRedirectOut, Value: ">"})
					}

			case char == '<' && !inQuote:
					if current.Len() > 0 {
							tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
							current.Reset()
					}
					tokens = append(tokens, Token{Type: TokenRedirectIn, Value: "<"})

			case char == '&' && !inQuote:
					if pos+1 < len(input) && input[pos+1] == '&' {
							if current.Len() > 0 {
									tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
									current.Reset()
							}
							tokens = append(tokens, Token{Type: TokenAnd, Value: "&&"})
							pos++ // Skip next character
					}

			case char == '$' && !inQuote:
					if current.Len() > 0 {
							tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
							current.Reset()
					}
					
					varName := p.extractVariableName(input[pos+1:])
					
					if varName != "" {
							tokens = append(tokens, Token{Type: TokenVariable, Value: varName})
							pos += len(varName)
					}

			case char == ';' && !inQuote:
					if current.Len() > 0 {
						tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
						current.Reset()
				}
				
				tokens = append(tokens, Token{Type: TokenSemicolon, Value: ";"})

		case char == ' ' && !inQuote:
				if current.Len() > 0 {
						tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
						current.Reset()
				}

		default:
				current.WriteRune(char)
		}
}

		if inQuote {
				return nil, &ParseError{Message: "unclosed quote", Pos: len(input)}
		}

		if current.Len() > 0 {
			tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
		}

	return tokens, nil
}

func (p *Parser) extractVariableName(input string) string {
		var name strings.Builder
		for i, c := range input {
				if i == 0 && !isAlpha(c) {
						return ""
				}
				
				if !isAlphaNumeric(c) && c != '_' {
						break
				}
		
				name.WriteRune(c)
		}
	
		return name.String()
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNumeric(c rune) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}

func (p *Parser) parseTokens(tokens []Token) ([]Command, error) {
		var commands []Command
		var currentCommand Command
		// var err error

		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			switch token.Type {
				case TokenWord:
					if currentCommand.Args == nil {
							currentCommand.Args = []string{token.Value}
					} else {
							currentCommand.Args = append(currentCommand.Args, token.Value)
					}

			case TokenPipe:
					if len(currentCommand.Args) == 0 {
							return nil, &ParseError{Message: "empty command before pipe", Pos: i}
					}
					
					commands = append(commands, currentCommand)
					currentCommand = Command{}

			case TokenRedirectIn:
					if i+1 >= len(tokens) {
							return nil, &ParseError{Message: "missing input file", Pos: i}
					}
					
					i++
					currentCommand.Stdin = tokens[i].Value

			case TokenRedirectOut, TokenRedirectAppend:
					if i+1 >= len(tokens) {
							return nil, &ParseError{Message: "missing output file", Pos: i}
				}
					
				i++
					currentCommand.Stdout = tokens[i].Value
					currentCommand.StdoutAppend = token.Type == TokenRedirectAppend

			case TokenVariable:
					value := os.Getenv(token.Value)
					
					if currentCommand.Args == nil {
							currentCommand.Args = []string{value}
					} else {
							currentCommand.Args = append(currentCommand.Args, value)
					}
				}
			}
		
			if len(currentCommand.Args) > 0 {
				commands = append(commands, currentCommand)
		}

		return commands, nil
}
