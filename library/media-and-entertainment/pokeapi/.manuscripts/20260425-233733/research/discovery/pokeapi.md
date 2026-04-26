# PokeAPI discovery notes

Official OpenAPI spec: https://raw.githubusercontent.com/PokeAPI/pokeapi/master/openapi.yml

The useful surface is not just endpoint lookup. PokeAPI is a graph: Pokemon -> species -> evolution chain; Pokemon -> types -> damage relations; Pokemon -> moves -> version-group details. Agent users ask graph questions, so the transcendence pass should expose graph commands.
