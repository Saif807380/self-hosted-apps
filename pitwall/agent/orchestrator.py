from langchain_google_genai import ChatGoogleGenerativeAI
from langgraph.prebuilt import create_react_agent

import pitwall.config.config as config
from agent.tools import ALL_TOOLS
from agent.prompts import build_system_prompt


HUMAN_MESSAGE = (
    "Before making any recommendations, you MUST call ALL of these tools to gather data:\n"
    "1. get_fantasy_state — current roster and market prices\n"
    "2. get_practice_telemetry — FP1/FP2 pace data\n"
    "3. get_season_form — season standings and form\n"
    "4. get_circuit_history_data — past results at this circuit\n"
    "5. get_weather_forecast — weekend weather\n"
    "6. get_recent_news — recent F1 news for penalties/injuries/upgrades\n\n"
    "Call all 6 tools first, then analyze the combined data to recommend the optimal roster "
    "that maximizes expected points while obeying all constraints. Show your reasoning."
)


def run_agent(verbose: bool = False, tools: list | None = None) -> str:
    llm = ChatGoogleGenerativeAI(
        model=config.GEMINI_MODEL,
        google_api_key=config.GEMINI_API_KEY,
        temperature=0.2,
    )

    agent_tools = tools if tools is not None else ALL_TOOLS
    agent = create_react_agent(llm, agent_tools)

    system_prompt = build_system_prompt()

    result = agent.invoke({
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": HUMAN_MESSAGE},
        ],
    })

    messages = result.get("messages", [])

    if verbose:
        return "\n\n---\n\n".join(
            getattr(m, "content", str(m))
            for m in messages
            if getattr(m, "content", None)
        )

    # Return only the final AI message
    for m in reversed(messages):
        content = getattr(m, "content", None)
        if content and hasattr(m, "type") and m.type == "ai":
            return content

    return "No recommendation generated. Check agent logs."
