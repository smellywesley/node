import os

from agents import Agent


def create_agent() -> Agent:
    return Agent(
        name="Live reviewed coding agent",
        instructions=(
            "You are running inside AgentOS. Return only the contents of a small "
            "reviewed Go source file. Do not include markdown fences, shell "
            "commands, explanations, or paths. The file must declare package "
            "reviewed and define Add(left int, right int) int."
        ),
        model=os.environ["OPENAI_MODEL"],
    )