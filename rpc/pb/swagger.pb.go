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
              "$ref": "#/definitions/rpcpbAccount"
            }
          }
        },
        "parameters": [
          {
            "name": "address",
            "description": "Send only one between address and alias\nHex string of the account addresss.",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "alias",
            "description": "String of the account alias.",
            "in": "query",
            "required": false,
            "type": "string"
          },
          {
            "name": "type",
            "description": "Send only one between type and height\nBlock type \"genesis\", \"confirmed\", or \"tail\".",
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
    "/v1/block": {
      "get": {
        "operationId": "GetBlock",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbBlock"
            }
          }
        },
        "parameters": [
          {
            "name": "hash",
            "description": "Send only one among hash, type and height\nBlock hash.",
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
              "$ref": "#/definitions/rpcpbBlocks"
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
              "$ref": "#/definitions/rpcpbCandidates"
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
              "$ref": "#/definitions/rpcpbDynasty"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/healthcheck": {
      "get": {
        "operationId": "HealthCheck",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbHealth"
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
              "$ref": "#/definitions/rpcpbMedState"
            }
          }
        },
        "tags": [
          "ApiService"
        ]
      }
    },
    "/v1/subscribe": {
      "get": {
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
            "name": "topics",
            "in": "query",
            "required": false,
            "type": "array",
            "items": {
              "type": "string"
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
              "$ref": "#/definitions/rpcpbTransaction"
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
              "$ref": "#/definitions/rpcpbTransactionHash"
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
    "/v1/transaction/receipt": {
      "get": {
        "operationId": "GetTransactionReceipt",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbTransactionReceipt"
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
      }
    },
    "/v1/transactions/pending": {
      "get": {
        "operationId": "GetPendingTransactions",
        "responses": {
          "200": {
            "description": "",
            "schema": {
              "$ref": "#/definitions/rpcpbTransactions"
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
    "rpcpbAccount": {
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
        "bandwidth": {
          "type": "string"
        },
        "unstaking": {
          "type": "string"
        },
        "alias": {
          "type": "string"
        },
        "candidate_id": {
          "type": "string"
        }
      }
    },
    "rpcpbBlock": {
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
        "hash_alg": {
          "type": "integer",
          "format": "int64",
          "title": "Block hash algorithm"
        },
        "crypto_alg": {
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
          "title": "Root hash of txs trie"
        },
        "dpos_root": {
          "type": "string",
          "title": "Root hash of dpos state trie"
        },
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbTransaction"
          },
          "title": "Transactions in block"
        },
        "tx_hashes": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      }
    },
    "rpcpbBlocks": {
      "type": "object",
      "properties": {
        "blocks": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbBlock"
          }
        }
      }
    },
    "rpcpbCandidate": {
      "type": "object",
      "properties": {
        "candidate_id": {
          "type": "string"
        },
        "address": {
          "type": "string"
        },
        "url": {
          "type": "string"
        },
        "collateral": {
          "type": "string"
        },
        "vote_power": {
          "type": "string"
        }
      }
    },
    "rpcpbCandidates": {
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
    "rpcpbDynasty": {
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
    "rpcpbHealth": {
      "type": "object",
      "properties": {
        "ok": {
          "type": "boolean",
          "format": "boolean"
        }
      }
    },
    "rpcpbMedState": {
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
        "lib": {
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
        "to": {
          "type": "string",
          "description": "Hex string of the sender account addresss."
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
        "hash_alg": {
          "type": "integer",
          "format": "int64",
          "title": "Transaction hash algorithm"
        },
        "crypto_alg": {
          "type": "integer",
          "format": "int64",
          "description": "Transaction crypto algorithm."
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
    },
    "rpcpbTransaction": {
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
        "hash_alg": {
          "type": "integer",
          "format": "int64",
          "title": "Transaction hash algorithm"
        },
        "crypto_alg": {
          "type": "integer",
          "format": "int64",
          "title": "Transaction crypto algorithm"
        },
        "sign": {
          "type": "string",
          "description": "Transaction sign."
        },
        "payer_sign": {
          "type": "string",
          "description": "Transaction payer's sign."
        },
        "on_chain": {
          "type": "boolean",
          "format": "boolean",
          "description": "If transaction is included in the block, it returns true. otherwise, false."
        },
        "receipt": {
          "$ref": "#/definitions/rpcpbTransactionReceipt",
          "title": "Transaction receipt"
        }
      }
    },
    "rpcpbTransactionHash": {
      "type": "object",
      "properties": {
        "hash": {
          "type": "string",
          "description": "Hex string of transaction hash."
        }
      }
    },
    "rpcpbTransactionReceipt": {
      "type": "object",
      "properties": {
        "executed": {
          "type": "boolean",
          "format": "boolean"
        },
        "cpu_usage": {
          "type": "string"
        },
        "net_usage": {
          "type": "string"
        },
        "error": {
          "type": "string"
        }
      }
    },
    "rpcpbTransactions": {
      "type": "object",
      "properties": {
        "transactions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/rpcpbTransaction"
          }
        }
      }
    }
  }
}
`
)
