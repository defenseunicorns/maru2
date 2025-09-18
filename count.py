# using this temporarily to roughly calculate tokens
# while i think about how to provide info from the docs within the MCP server
#
# usage: python3 count.py docs/*.md
import tiktoken
import sys

def count_tokens_in_file(filepath, model="gpt-4o-mini"):
    enc = tiktoken.encoding_for_model(model)
    with open(filepath, "r", encoding="utf-8") as f:
        text = f.read()
    return len(enc.encode(text))

if __name__ == "__main__":
    filepaths = sys.argv[1:]
    for fp in filepaths:
        print(f"{fp} {count_tokens_in_file(fp)}")
