import tiktoken
import sys

def count_tokens_in_file(filepath, model="gpt-4o-mini"):
    enc = tiktoken.encoding_for_model(model)
    with open(filepath, "r", encoding="utf-8") as f:
        text = f.read()
    return len(enc.encode(text))

if __name__ == "__main__":
    filepath = sys.argv[1]
    print(count_tokens_in_file(filepath))
