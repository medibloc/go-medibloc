package rpcpb

const (
	swagger = `{
  "swagger": "2.0",
  "info": {
    "title": "rpc.proto",
    "version": "version not set"
  },
  "schemes": [
    "http",
    "https"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/v1/block": {
      "get": {
        "operationId": "GetBlock",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbBlockResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "hash",
            "description": "Block hash. Or the string \"genesis\", \"confirmed\", \"tail\".",
            "in": "query",
            "required": false,
            "type": "string"
          }
        ],
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/blocks": {
      "get": {
        "operationId": "GetBlocks",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbBlocksResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "from",
            "in": "query",
            "required": false,
            "type": "string",
            "format": "uint64"
          },
          {
            "name": "to",
            "in": "query",
            "required": false,
            "type": "string",
            "format": "uint64"
          }
        ],
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/node/medstate": {
      "get": {
        "operationId": "GetMedState",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetMedStateResponse"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/subscribe": {
      "post": {
        "operationId": "Subscribe",
        "responses": {
          "200": {
            "description": "(streaming responses)",
            "schema": {
              "$ref": "#/definitions/rpcpbSubscribeResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/rpcpbSubscribeRequest"
            }
          }
        ],
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/transaction": {
      "get": {
        "operationId": "GetTransaction",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbTransactionResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "hash",
            "description": "Transaction hash.",
            "in": "query",
            "required": false,
            "type": "string"
          }
        ],
        "tags": [
          "ApiService"
        ]
      },
      "post": {
        "operationId": "SendTransaction",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbSendTransactionResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/rpcpbSendTransactionRequest"
            }
          }
        ],
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/transactions/pending": {
      "get": {
        "operationId": "GetPendingTransactions",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbTransactionsResponse"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/user/accounts": {
      "get": {
        "operationId": "GetAccounts",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbAccountsResponse"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/user/accountstate": {
      "get": {
        "operationId": "GetAccountState",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetAccountStateResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "address",
            "description": "Hex string of the account addresss.",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "height",
            "description": "block account state with height. Or the string \"genesis\", \"confirmed\", \"tail\".",
            "in": "query",
            "required": false,
            "type": "string"
          }
        ],
        "tags": [
          "ApiService"
        ]
      }
    }
  },
  "definitions": {
    "rpcpbAccountsResponse": {
      "type": "object",
      "properties": {
        "accounts": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbGetAccountStateResponse"
          }
        }
      }
    },
    "rpcpbBlockResponse": {
      "type": "object",
      "properties": {
        "hash": {
          "type": "string",
          "title": "Block hash"
        },
        "parent_hash": {
          "type": "string",
          "title": "Block parent hash"
        },
        "coinbase": {
          "type": "string",
          "title": "Block coinbase address"
        },
        "timestamp": {
          "type": "string",
          "format": "int64",
          "title": "Block timestamp"
        },
        "chain_id": {
          "type": "integer",
          "format": "int64",
          "title": "Block chain id"
        },
        "alg": {
          "type": "integer",
          "format": "int64",
          "title": "Block signature algorithm"
        },
        "sign": {
          "type": "string",
          "title": "Block signature"
        },
        "accs_root": {
          "type": "string",
          "title": "Root hash of accounts trie"
        },
        "txs_root": {
          "type": "string",
          "title": "Root hash of transactions trie"
        },
        "usage_root": {
          "type": "string",
          "title": "Root hash of usage trie"
        },
        "records_root": {
          "type": "string",
          "title": "Root hash of records trie"
        },
        "consensus_root": {
          "type": "string",
          "title": "Root hash of consensus trie"
        },
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbTransactionResponse"
          },
          "title": "Transactions in block"
        },
        "height": {
          "type": "string",
          "format": "uint64",
          "title": "Block height"
        }
      }
    },
    "rpcpbBlocksResponse": {
      "type": "object",
      "properties": {
        "blocks": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbBlockResponse"
          }
        }
      }
    },
    "rpcpbGetAccountStateResponse": {
      "type": "object",
      "properties": {
        "address": {
          "type": "string",
          "description": "Hex string of the account address."
        },
        "balance": {
          "type": "string",
          "description": "Current balance in unit of 1/(10^8) MED."
        },
        "certs_issued": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Account addresses certificated by the account."
        },
        "certs_received": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Account addresses that have certificated the account."
        },
        "nonce": {
          "type": "string",
          "format": "uint64",
          "description": "Current transaction count."
        },
        "records": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "List of record hash."
        },
        "vesting": {
          "type": "string",
          "description": "Current vesting in unit of 1/(10^8) MED."
        },
        "voted": {
          "type": "string",
          "description": "Voted address."
        },
        "txs_send": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "title": "Transactions sent from account"
        },
        "txs_get": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "title": "Transactions sent to account"
        }
      }
    },
    "rpcpbGetMedStateResponse": {
      "type": "object",
      "properties": {
        "chain_id": {
          "type": "integer",
          "format": "int64",
          "title": "Block chain id"
        },
        "tail": {
          "type": "string",
          "title": "Current tail block hash"
        },
        "height": {
          "type": "string",
          "format": "uint64",
          "title": "Current tail block height"
        },
        "LIB": {
          "type": "string",
          "title": "Current LIB hash"
        }
      }
    },
    "rpcpbSendTransactionRequest": {
      "type": "object",
      "properties": {
        "hash": {
          "type": "string",
          "title": "Transaction hash"
        },
        "from": {
          "type": "string",
          "description": "Hex string of the sender account addresss."
        },
        "to": {
          "type": "string",
          "description": "Hex string of the receiver account addresss."
        },
        "value": {
          "type": "string",
          "description": "Amount of value sending with this transaction."
        },
        "timestamp": {
          "type": "string",
          "format": "int64",
          "description": "Transaction timestamp."
        },
        "data": {
          "$ref": "#/definitions/rpcpbTransactionData",
          "description": "Transaction Data type."
        },
        "nonce": {
          "type": "string",
          "format": "uint64",
          "description": "Transaction nonce."
        },
        "chain_id": {
          "type": "integer",
          "format": "int64",
          "description": "Transaction chain ID."
        },
        "alg": {
          "type": "integer",
          "format": "int64",
          "description": "Transaction algorithm."
        },
        "sign": {
          "type": "string",
          "description": "Transaction sign."
        },
        "payer_sign": {
          "type": "string",
          "description": "Transaction payer's sign."
        }
      }
    },
    "rpcpbSendTransactionResponse": {
      "type": "object",
      "properties": {
        "hash": {
          "type": "string",
          "description": "Hex string of transaction hash."
        }
      }
    },
    "rpcpbSubscribeRequest": {
      "type": "object",
      "properties": {
        "topics": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      }
    },
    "rpcpbSubscribeResponse": {
      "type": "object",
      "properties": {
        "topic": {
          "type": "string"
        },
        "data": {
          "type": "string"
        }
      }
    },
    "rpcpbTransactionData": {
      "type": "object",
      "properties": {
        "type": {
          "type": "string",
          "description": "Transaction data type."
        },
        "payload": {
          "type": "string",
          "description": "Transaction data payload."
        }
      }
    },
    "rpcpbTransactionResponse": {
      "type": "object",
      "properties": {
        "hash": {
          "type": "string",
          "title": "Transaction hash"
        },
        "from": {
          "type": "string",
          "description": "Hex string of the sender account addresss."
        },
        "to": {
          "type": "string",
          "description": "Hex string of the receiver account addresss."
        },
        "value": {
          "type": "string",
          "description": "Amount of value sending with this transaction."
        },
        "timestamp": {
          "type": "string",
          "format": "int64",
          "description": "Transaction timestamp."
        },
        "data": {
          "$ref": "#/definitions/rpcpbTransactionData",
          "description": "Transaction Data type."
        },
        "nonce": {
          "type": "string",
          "format": "uint64",
          "description": "Transaction nonce."
        },
        "chain_id": {
          "type": "integer",
          "format": "int64",
          "description": "Transaction chain ID."
        },
        "alg": {
          "type": "integer",
          "format": "int64",
          "description": "Transaction algorithm."
        },
        "sign": {
          "type": "string",
          "description": "Transaction sign."
        },
        "payer_sign": {
          "type": "string",
          "description": "Transaction payer's sign."
        },
        "executed": {
          "type": "boolean",
          "format": "boolean",
          "description": "If transaction is executed and included in the block, it returns true. otherwise, false."
        }
      }
    },
    "rpcpbTransactionsResponse": {
      "type": "object",
      "properties": {
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbTransactionResponse"
          }
        }
      }
    }
  }
}
`
)
