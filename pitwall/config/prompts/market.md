I'm on the F1 Fantasy create-team page at https://fantasy.formula1.com/en/create-team.

Please do the following:

Read the Drivers tab — scroll through the full list to capture every driver's name, season points, and value (price in $M).

Switch to the Constructors tab — scroll through the full list to capture every constructor's name, season points, and value (price in $M).

Compile all the data into the following JSON format and give it to me in the chat so I can save it as market.json:

```json
{
  "market": [
    {"name": "M. Verstappen", "type": "driver", "price": 27.7, "points_scored": -5},
    ...
    {"name": "Red Bull Racing", "type": "constructor", "price": 28.2, "points_scored": 8}
  ]
}
```

Rules:

type: "driver" or "constructor".

price: Numeric value in millions (e.g. $27.7M → 27.7).

points_scored: Integer, include negatives where applicable.

Include all drivers and constructors — do not skip any.

