import itertools

coins = {
	'corroded': 3,
	'red': 2,
	'shiny': 5,
	'concave': 7,
	'blue': 9
}

for option in itertools.permutations(coins):
	calc = coins[option[0]] + coins[option[1]] * coins[option[2]]**2 + coins[option[3]]**3 - coins[option[4]]
	if calc == 399:
		print option
		break
