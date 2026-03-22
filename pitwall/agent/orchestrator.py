from langchain_google_genai import ChatGoogleGenerativeAI
from langgraph.prebuilt import create_react_agent

import config.config as config
from agent.tools import ALL_TOOLS
from agent.prompts import build_system_prompt


def _build_human_message() -> str:
    tools = [
        "1. get_fantasy_state — current roster and market prices",
        "2. get_practice_telemetry — FP1 pace data" if config.IS_SPRINT_WEEKEND else "2. get_practice_telemetry — FP1/FP2 pace data",
    ]
    if config.IS_SPRINT_WEEKEND:
        tools.append("3. get_sprint_telemetry — sprint qualifying grid positions and best lap times")
        tools += [
            "4. get_season_form — season standings and form",
            "5. get_circuit_history_data — past results at this circuit",
            "6. get_weather_forecast — weekend weather",
            "7. get_recent_news — recent F1 news for penalties/injuries/upgrades",
        ]
    else:
        tools += [
            "3. get_season_form — season standings and form",
            "4. get_circuit_history_data — past results at this circuit",
            "5. get_weather_forecast — weekend weather",
            "6. get_recent_news — recent F1 news for penalties/injuries/upgrades",
        ]
    tool_list = "\n".join(tools)
    count = len(tools)
    return (
        f"Before making any recommendations, you MUST call ALL of these tools to gather data:\n"
        f"{tool_list}\n\n"
        f"Call all {count} tools first, then use the combined data to answer the request below. "
        "Show your reasoning."
    )


HUMAN_MESSAGE = _build_human_message()

DEFAULT_TASK = (
    "Recommend the optimal roster that maximizes expected points while obeying all constraints."
)


def run_agent(verbose: bool = False, tools: list | None = None, query: str | None = None) -> str:
    llm = ChatGoogleGenerativeAI(
        model=config.GEMINI_MODEL,
        google_api_key=config.GEMINI_API_KEY,
        temperature=0.2,
    )

    agent_tools = tools if tools is not None else ALL_TOOLS
    agent = create_react_agent(llm, agent_tools)

    system_prompt = build_system_prompt(query=query)

    task = query.strip() if query else DEFAULT_TASK
    user_message = f"{HUMAN_MESSAGE}\n\n**Your task:** {task}"

    result = agent.invoke({
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_message},
        ],
    })

    messages = result.get("messages", [])

    if verbose:
        parts = []
        for m in messages:
            content = getattr(m, "content", None)
            if not content:
                continue
            if isinstance(content, list):
                text = " ".join(
                    c.get("text", "") if isinstance(c, dict) else str(c)
                    for c in content
                )
            else:
                text = str(content)
            if text.strip():
                parts.append(text)
        return "\n\n---\n\n".join(parts)

    # Return only the final AI message
    for m in reversed(messages):
        content = getattr(m, "content", None)
        if content and hasattr(m, "type") and m.type == "ai":
            if isinstance(content, list):
                return " ".join(
                    c.get("text", "") if isinstance(c, dict) else str(c)
                    for c in content
                )
            return content

    return "No recommendation generated. Check agent logs."
