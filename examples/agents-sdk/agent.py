import os

from agents import Agent


def create_agent() -> Agent:
    return Agent(
        name="Word Count Assistant",
        instructions=(
            "Help with text-analysis tasks. Count words directly and return "
            "a concise answer."
        ),
        model=os.environ["OPENAI_MODEL"],
    )
