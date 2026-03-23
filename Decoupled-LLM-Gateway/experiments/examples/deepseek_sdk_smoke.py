#!/usr/bin/env python3
"""Optional: call DeepSeek via OpenAI SDK (no gateway). pip3 install openai"""

import os

from openai import OpenAI


def main() -> None:
    key = os.environ.get("DEEPSEEK_API_KEY")
    if not key:
        raise SystemExit("set DEEPSEEK_API_KEY")
    client = OpenAI(api_key=key, base_url="https://api.deepseek.com")
    response = client.chat.completions.create(
        model="deepseek-chat",
        messages=[
            {"role": "system", "content": "You are a helpful assistant"},
            {"role": "user", "content": "Hello"},
        ],
        stream=False,
    )
    print(response.choices[0].message.content)


if __name__ == "__main__":
    main()
