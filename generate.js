const fs = require('fs')
const Big = require('big.js')

const IS_TESTNET = true
const TESTNET_CHAINID = "epix_4243-1"
const MAINNET_CHAINID = "epix_4242-1"
const CSV_FILE_PAHT = "./snapshot/testnet/snapshot.csv"
const FUNDS_POOL_AMOUNT = 23668256.824195825

const decimal = 18
const KEY = "mykey"
const MONIKER = "localtestnet"
const KEYRING = "test"
const KEYALGO = "eth_secp256k1"
const LOGLEVEL = "info"
const TRACE = ""

const calculateTokenAmount = (amount) => {
    let str = amount.toString()
    let intStr = str.split('.')[0]
    let floatStr = str.split('.')[1] || ""

    intStr = intStr.replace(/^0+/, '');

    let endStr = ''
    for(let i = 0; i < decimal - floatStr.length; i ++) {
        endStr += "0"
    }

    if(intStr.length == 0) floatStr = floatStr.replace(/^0+/, '');
    
    let resultStr = intStr + floatStr + endStr
    return resultStr
}

const generate = async () => {

    try {
        let content = ''

        content += `KEY="${KEY}"
CHAINID="${IS_TESTNET ? TESTNET_CHAINID : MAINNET_CHAINID}"
MONIKER="${MONIKER}"
KEYRING="${KEYRING}"
KEYALGO="${KEYALGO}"
LOGLEVEL="${LOGLEVEL}"
# to trace evm
#TRACE="--trace"
TRACE="${TRACE}"\n\n`

        content += `# validate dependencies are installed\n`
        content += `command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }\n\n`

        content += `# Reinstall daemon\n`
        content += `rm -rf ~/.epixd*\n`
        content += `make install\n\n\n\n`

        content += `# Set client config\n`
        content += `epixd config set client chain-id $CHAINID\n`
        content += `epixd config set client keyring-backend $KEYRING\n\n`

        content += `# if $KEY exists it should be deleted\n`
        content += `epixd keys add $KEY --keyring-backend $KEYRING --algo $KEYALGO\n\n`

        content += `# Set moniker and chain-id for epix (Moniker can be anything, chain-id must be an integer)\n`
        content += `epixd init $MONIKER --chain-id $CHAINID\n\n`

        content += `# Change parameter token denominations to aepix\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["staking"]["params"]["bond_denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["crisis"]["constant_fee"]["denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["coinswap"]["params"]["pool_creation_fee"]["denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["coinswap"]["standard_denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["gov"]["params"]["expedited_min_deposit"][0]["denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["evm"]["params"]["evm_denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.app_state["inflation"]["params"]["mint_denom"]="aepix"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n\n`

        content += `# Set gas limit in genesis\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq '.consensus["params"]["block"]["max_gas"]="10000000"' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n\n`

        content += `# disable produce empty block\n`
        content += `if [[ "$OSTYPE" == "darwin"* ]]; then\n`
        content += `    sed -i '' 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.epixd/config/config.toml\n`
        content += `  else\n`
        content += `    sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.epixd/config/config.toml\n`
        content += `fi\n\n`

        content += `if [[ $1 == "pending" ]]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.epixd/config/config.toml
        sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.epixd/config/config.toml
    else
        sed -i 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.epixd/config/config.toml
        sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.epixd/config/config.toml
    fi
fi\n\n\n`

        let totalSupply = 0
        const KEY_AEPIX_AMOUNT = (FUNDS_POOL_AMOUNT + 1)
        totalSupply += (FUNDS_POOL_AMOUNT + 1)
        totalSupply = new Big(totalSupply)
        console.log('Total supply:', totalSupply.toString())
        content += `epixd add-genesis-account $KEY ${calculateTokenAmount(KEY_AEPIX_AMOUNT)}aepix --keyring-backend $KEYRING\n\n`

        try {
            const csvContent = fs.readFileSync(CSV_FILE_PAHT, 'utf-8')
            const lines = csvContent.split('\n')
            for (let i = 0; i < lines.length; i++) {
                const line = lines[i]
                const account = line.split(",")[0]
                const amount = line.split(",")[1].toString()
                console.log('amount:', parseFloat(amount))
                content += `epixd add-genesis-account ${account} ${calculateTokenAmount(amount)}aepix\n`
                totalSupply = totalSupply.plus(new Big(amount))
                console.log('Total supply:', totalSupply.toString())
            }
        } catch (error) {

        }
        content += `
        # Update total supply with claim values
        validators_supply=$(cat $HOME/.epixd/config/genesis.json | jq -r '.app_state["bank"]["supply"][0]["amount"]')
        `
        content += `total_supply=${calculateTokenAmount(totalSupply)}\n`
        content += `cat $HOME/.epixd/config/genesis.json | jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' > $HOME/.epixd/config/tmp_genesis.json && mv $HOME/.epixd/config/tmp_genesis.json $HOME/.epixd/config/genesis.json\n`

        content += `
        # Sign genesis transaction
        epixd gentx $KEY ${1 * 10 ** decimal}aepix --keyring-backend $KEYRING --chain-id $CHAINID
        `

        content += `
        # Collect genesis tx
        epixd collect-gentxs
        `

        content += `
        # Run this to ensure everything worked and that the genesis file is setup correctly
        epixd validate-genesis
        `

        content += `
        if [[ $1 == "pending" ]]; then
          echo "pending mode is on, please wait for the first block committed."
        fi
        `

        content += `
        # Start the node (remove the --pruning=nothing flag if historical queries are not needed)
        epixd start --pruning=nothing $TRACE --log_level $LOGLEVEL --minimum-gas-prices=0.0001aepix --json-rpc.api eth,txpool,personal,net,debug,web3 --rpc.laddr "tcp://0.0.0.0:26657" --api.enable --chain-id $CHAINID
        epixd tx distribution fund-community-pool ${calculateTokenAmount(FUNDS_POOL_AMOUNT)}aepix --from $KEY --gas-prices 100aepix  --keyring-backend $KEYRING --chain-id $CHAINID
        `

        fs.writeFileSync('./init.sh', content, 'utf-8')
    } catch (error) {
        console.log(error)
    }   

}

generate()
