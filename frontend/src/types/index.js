/**
 * Shared shapes returned by the HNL Wallet API. Plain JSDoc typedefs — this
 * project uses JavaScript, not TypeScript, but keeping the response shapes
 * documented in one place still pays off for autocomplete and readability.
 *
 * @typedef {Object} User
 * @property {string} id
 * @property {string} email
 * @property {string} full_name
 * @property {string} created_at
 *
 * @typedef {Object} LoginResponse
 * @property {string} token
 * @property {{ id: string, email: string, full_name: string }} user
 *
 * @typedef {Object} AccountSummary
 * @property {string} account_number
 * @property {string} account_type
 * @property {string} currency
 * @property {number} balance
 *
 * @typedef {Object} AccountsSummary
 * @property {number} total_balance
 * @property {string} currency
 * @property {AccountSummary[]} accounts
 *
 * @typedef {Object} AccountDetail
 * @property {string} account_number
 * @property {string} currency
 * @property {string} account_type
 * @property {number} initial_balance
 * @property {number} balance
 *
 * @typedef {Object} Transaction
 * @property {string} id
 * @property {string} type
 * @property {"incoming"|"outgoing"} direction
 * @property {number} amount
 * @property {string} counterparty_account_number
 * @property {string} timestamp
 *
 * @typedef {Object} PendingAction
 * @property {string} tool
 * @property {Object<string, any>} arguments
 *
 * @typedef {Object} ChatResponse
 * @property {string} reply
 * @property {boolean} [requires_confirmation]
 * @property {string} [confirmation_token]
 * @property {PendingAction} [pending_action]
 */

export {};
