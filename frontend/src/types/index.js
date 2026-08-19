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
 */

export {};
