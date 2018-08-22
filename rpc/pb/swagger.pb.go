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
    "/v1/account": {
      "get": {
        "operationId": "GetAccount",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetAccountResponse"
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
            "name": "type",
            "description": "If you send type, height field is ignored.\nBlock type \"genesis\", \"confirmed\", or \"tail\".",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "height",
            "description": "Block account state with height.",
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
    "/v1/account/{address}/transactions": {
      "get": {
        "operationId": "GetAccountTransactions",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetTransactionsResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "address",
            "description": "Hex string of the account addresss.",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "include_pending",
            "description": "Whether or not to include pending transactions. Default is true.",
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
    "/v1/block": {
      "get": {
        "operationId": "GetBlock",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetBlockResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "hash",
            "description": "If you send hash, type and height field is ignored.\nBlock hash.",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "type",
            "description": "Block type \"genesis\", \"confirmed\", or \"tail\".",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "height",
            "description": "Block height.",
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
    "/v1/blocks": {
      "get": {
        "operationId": "GetBlocks",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetBlocksResponse"
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
    "/v1/candidates": {
      "get": {
        "operationId": "GetCandidates",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetCandidatesResponse"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/dynasty": {
      "get": {
        "operationId": "GetDynasty",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbGetDynastyResponse"
            }
          }
        },
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
              "$ref": "#/definitions/rpcpbGetTransactionResponse"
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
              "$ref": "#/definitions/rpcpbGetTransactionsResponse"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    }
  },
  "definitions": {
    "rpcpbCandidate": {
      "type": "object",
      "properties": {
        "address": {
          "type": "string"
        },
        "collatral": {
          "type": "string"
        },
        "votePower": {
          "type": "string"
        }
      }
    },
    "rpcpbGetAccountResponse": {
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
        "nonce": {
          "type": "string",
          "format": "uint64",
          "description": "Current transaction count."
        },
        "vesting": {
          "type": "string",
          "description": "Current vesting in unit of 1/(10^8) MED."
        },
        "voted": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Voted address."
        },
        "records": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "List of record hash."
        },
        "certs_received": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Account addresses that have certificated the account."
        },
        "certs_issued": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Account addresses certificated by the account."
        },
        "txs_from": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "title": "Transactions sent from account"
        },
        "txs_to": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "title": "Transactions sent to account"
        }
      }
    },
    "rpcpbGetBlockResponse": {
      "type": "object",
      "properties": {
        "height": {
          "type": "string",
          "format": "uint64",
          "title": "Block height"
        },
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
        "reward": {
          "type": "string",
          "title": "Block reward"
        },
        "supply": {
          "type": "string",
          "title": "Block supply"
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
        "data_root": {
          "type": "string",
          "title": "Root hash of data trie"
        },
        "dpos_root": {
          "type": "string",
          "title": "Root hash of dpos state trie"
        },
        "usage_root": {
          "type": "string",
          "title": "Root hash of usage trie"
        },
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbGetTransactionResponse"
          },
          "title": "Transactions in block"
        }
      }
    },
    "rpcpbGetBlocksResponse": {
      "type": "object",
      "properties": {
        "blocks": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbGetBlockResponse"
          }
        }
      }
    },
    "rpcpbGetCandidatesResponse": {
      "type": "object",
      "properties": {
        "candidates": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbCandidate"
          }
        }
      }
    },
    "rpcpbGetDynastyResponse": {
      "type": "object",
      "properties": {
        "addresses": {
          "type": "array",
          "items": {
            "type": "string"
          }
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
    "rpcpbGetTransactionResponse": {
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
        "tx_type": {
          "type": "string",
          "description": "Transaction type."
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
        "payload": {
          "type": "string",
          "description": "Transaction payload."
        },
        "alg": {
          "type": "integer",
          "format": "int64",
          "title": "Transaction algorithm"
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
    "rpcpbGetTransactionsResponse": {
      "type": "object",
      "properties": {
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbGetTransactionResponse"
          }
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
        "tx_type": {
          "type": "string",
          "description": "Transaction type."
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
        "payload": {
          "type": "string",
          "title": "Transaction payload"
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
        "hash": {
          "type": "string"
        }
      }
    }
  }
}
`
)
