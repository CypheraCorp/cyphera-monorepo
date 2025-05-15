# HIFI Bridge Developer Guide

*Generated on 5/13/2025*

## Overview

HIFI is a developer platform that facilitates money movement between stablecoins and fiat currencies across borders. Our platform and APIs enable businesses and developers to build secure and comprehensive financial applications with the internet as the backbone, connecting stablecoins and fiat currencies globally and offering tools for seamless and secure money movement.

### With our APIs, you can:

- **Onramp**: Convert fiat currencies into stablecoins, regardless of the fiat currencies you are using.
  - **Link Bank Accounts**: Connect users' bank accounts to make onramping quick and easy.
  - **Virtual Accounts**: Generate virtual accounts for the users where deposited fiat is instantly converted into stablecoins and transferred to the user's wallet on chain.

- **Offramp**: Convert stablecoins into fiat currencies, no matter where users are located.
  - **Link Bank Accounts**: Connect users' bank accounts for smooth offramp.
  - **Liquidation Addresses**: Generate liquidation wallet addresses where received stablecoins are instantly converted to fiat and deposited into the user's bank account.

- **Provision Wallets**: Provision and manage wallets for individuals and businesses, ensuring their on-chain funds are safe and easily accessible.

- **Transfer Stablecoins**: Transfer stablecoin between user wallets.

- **KYC Compliance**: Leave compliance to us. Our in-house compliance team handles all KYC processes, ensuring your users meet regulatory requirements while relieving you of the compliance burden.

- **Bridge Across Chains**: Allows users to transfer stablecoins across blockchains.

- **Multi-Currency Account (MCA)**: Open Multi-Currency Accounts that allow users to hold, manage, exchange, and transact in multiple currencies.

## Key Concepts

The following are API terms and concepts:

### API
| Term | Definition |
|------|------------|
| API Key | A unique identifier used to authenticate and authorize requests to the API. |
| Idempotency Key | A unique key used to ensure that a request is processed only once to prevent duplicates. |

### Compliance & Identity Verification
| Term | Definition |
|------|------------|
| KYC | Stands for "Know Your Customer". This process verifies the identities of individuals and businesses to meet legal requirements and protect financial institutions against fraud, corruption, money laundering and terrorist financing. |
| Terms of Service | A legal agreement between HIFI and its users outlining the rules, responsibilities, and acceptable use of HIFI's platform. Users are required to review and accept the Terms of Service to ensure they understand and comply with these terms before creating a user and use any of the services we provide. |

### Accounts & Banking
| Term | Definition |
|------|------------|
| Onramp Account | A bank account used as the source for the onramping process. For example, during onramping, fiat currency from the onramp account is converted into stablecoin. |
| Offramp Account | A bank account used as the destination for the offramping process. For example, during offramping, stablecoin is converted into fiat currency and deposited into the offramp account. |
| Virtual Account | A virtual onramp bank account generate by our system to facilitate onramping. Users can transfer fiat into the virtual account, and the deposited fiat is instantly converted into stablecoins and transferred to the user's wallet on chain. |
| Liquidation Address | A wallet address generate by our system to facilitate offramping. Any stablecoins received by the liquidation address will be converted to fiat and deposited into the user's offramp bank account. |
| ACH (Automated Clearing House) | A network for electronic funds transfers and payments in the USA. |
| SEPA (Single Euro Payments Area) | An EU initiative that simplifies bank transfers in Euro. |
| PIX | A Brazilian instant payment system that allows for quick and efficient electronic transfers between bank accounts. |
| Spei | A large-value funds transfer system in Mexico. |

### Stablecoin & Blockchain
| Term | Definition |
|------|------------|
| Onramp | The conversion of fiat currency to crypto currency (eg. USD -> USDC) |
| Offramp | The conversion of crypto currency to fiat currency (eg. USDC -> USD) |
| Wallet Address | A blockchain address used to send and receive cryptocurrencies. It uniquely identifies a user's account on the blockchain. |
| Chain | A blockchain or distributed ledger that records and verifies transactions in a secure, decentralized manner. Eg. Ethereum mainnet is the primary network where Ethereum cryptocurrency transactions occur between users. |
| Stablecoin | A type of cryptocurrency designed to maintain a stable value relative to a fiat currency (e.g., USD) or other assets. Stablecoins aim to provide the benefits of digital currencies while minimizing the volatility typically associated with cryptocurrencies. For example, USD Coin (USDC) is a widely used stablecoin pegged to the US Dollar and is 1:1 backed by reserves of USD or equivalent assets to ensure price stability. |

## Sandbox Environment

Our sandbox environment allows you to test from user creation, KYC process, to adding onramp and/or offramp accounts. Sandbox mode is provided so that you can test your integration without incurring any usage charges. Transfers are not performed within sandbox mode.

Create an account at in the developer dashboard via the link provided to you by the sales team. Under the API Key header tab, "New API Key" button to configure and create your API key.

### Notes about HIFI's Sandbox Environment

Sandbox has a few key areas in which it differs from Production:
- User KYC will be automatically approved upon submitting dummy personal information (no real KYC/KYB process involved).
- There is no real money movement in Sandbox.
- The wallet addresses are created for testnets. A wallet address is a unique identifier that allows for the sending and receiving of cryptocurrencies. A testnet is a network that's similar to the mainnet but is used for testing and experimenting without real money movements.

If you have suggests for how to improve our sandbox experience, please reach out to us at techsupport@hifibridge.com, thanks!

## Quickstart Guide (Sandbox)

A quick introduction to building with HIFI API in sandbox environment.

### Welcome to the HIFI Quickstart Guide!

In this guide, we will walk you through the process of creating a user, passing KYC to unlock rails, adding onramp/offramp accounts, doing onramps, crypto transfers, and offramps. By following along, you'll gain a clear understanding of how our endpoints work and how to integrate them into your application.

To get started, you'll need an API key, which you can get by reaching out to a member of the sales team.

You'll have two different API keys for two different HIFI environments. Today we'll start in the Sandbox environment with the sandbox API key. You can follow our Sandbox guide to generate a sandbox API key.

### Let's first create a user!

#### User Introduction

A user object can represent either an individual or a business. All the available rails, accounts, onramps, offramps, transfer, etc, are associated with a user object.

To create a HIFI user, you need to provide a basic set of user information. The user must also review and accept HIFI's Terms and Service to obtain a valid signed agreement ID, which signifies a legally binding agreement to use our service.

A successfully created user will have provisioned wallets and be granted access to our on-chain functionalities.

#### Get a Valid Signed Agreement ID

To get HIFI's Terms and Service page, you can call the Generate Terms of Service Link endpoint. You will need to pass in an idempotencyKey, which can be any UUID. This idempotencyKey will be used as your signed agreement ID.

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/tos-link \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"idempotencyKey": "8cb49537-bcf9-41b1-8d8c-c9c200d7341b",
"redirectUrl": "http://redirect.url.com/tosredirect"
}
'
```

**Response:**
```json
{
"url": "https://dashboard.hifibridge.com/accept-terms-of-service?sessionToke...",
"signedAgreementId": "8cb49537-bcf9-41b1-8d8c-c9c200d7341b"
}
```

You will get a response object back containing the url and the signedAgreementId. The url directs you to HIFI's Terms of Service page. The signedAgreementId is the idempotencyKey you passed in, which will be valid only after you accept the Terms of Service.

#### Create User

Now that we have the user's valid signedAgreementId and basic personal information, we can create a user by calling the Create User endpoint. We have provided all the basic personal information with dummy values in the curl request.

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"type": "individual",
"firstName": "Post",
"lastName": "Man",
"email": "postman@gmail.com",
"dateOfBirth": "1997-02-17",
"address": {
"addressLine1": "Example St 1.",
"addressLine2": "Apt 123",
"city": "Hoboken",
"stateProvinceRegion": "NJ",
"postalCode": "07030",
"country": "USA"
},
"signedAgreementId": "8cb49537-bcf9-41b1-8d8c-c9c200d7341b"
}'
```

**Response:**
```json
{
"id": "f4fd2f10-2577-45cc-9e06-07ac9a74eb51",
"type": "individual",
"email": "postman@gmail.com",
"name": "Post Man",
"wallets": {
"INDIVIDUAL": {
"POLYGON_AMOY": {
"address": "0xa99F0308604Af7526Bb69FD0F292993B948161b5"
}
}
}
}
```

Let's take a moment to understand the response:
- The id is the user ID, which should be saved for future API calls for this particular user.
- The wallets object contains all the wallet types and addresses provisioned for the user.

We have successfully created a user with the user id: f4fd2f10-2577-45cc-9e06-07ac9a74eb51.

### KYC

After successfully creating a user, the user needs to decide which rails to unlock and submit KYC to enable access to those rails. We currently support the following rails:
- USD_EURO: Onramp and offramp services for USD and EURO.
- SOUTH_AMERICA_LIGHT: Offramp services for the South American region, including BRL, MXN, COP, and ARS, with a lower transaction limit.
- SOUTH_AMERICA_STANDARD: Offramp services for the South American region, including BRL, MXN, COP, and ARS, with a higher transaction limit.

To read more about the rails we support, click [here](#).
To read more about KYC in detail, click [here](#).

All rails can be unlocked through our KYC endpoints. To unlock, for example, the USD_EURO rail, follow these steps:
1. Gather the required KYC information for the USD_EURO rail by either consulting our KYC documentation or using the Retrieve KYC Requirements endpoint.
2. Update the user's KYC information using the Update KYC Information endpoint.
3. Submit the user's KYC information through the Submit KYC endpoint to unlock the rail.
4. Check the user's KYC status for the USD_EURO rail via the Retrieve KYC Status endpoint.

Let's go through each of these steps in detail to unlock the USD_EURO rail.

#### Retrieve KYC Requirements

The Retrieve KYC Requirements endpoint provides the required and optional KYC fields needed to unlock a specific rail, as well as any invalid or missing KYC information the user currently holds, assuming they intend to submit KYC for this rail. This information helps identify what needs to be updated via the Update KYC endpoint before submitting KYC for the rail. By using this helper endpoint, you can programmatically retrieve details that are also available in our KYC documentation.

Let's get the KYC requirements for USD_EURO rails by calling the Retrieve KYC Requirements endpoint and passing in USD_EURO as the rails and the userId.

**Request:**
```bash
curl --request GET \
--url 'https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/kyc/requirements?rails=USD_EURO' \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456'
```

**Response:**
```json
{
"userId": "f4fd2f10-2577-45cc-9e06-07ac9a74eb51",
"rails": "USD_EURO",
"type": "individual",
"required": {
"firstName": "string",
"lastName": "string",
"email": "string",
"phone": "string",
"address": {
"required": {
"addressLine1": "string",
"city": "string",
"stateProvinceRegion": "string",
"postalCode": "string",
"country": "string"
},
"optional": {
"addressLine2": "string"
}
},
"dateOfBirth": "date",
"taxIdentificationNumber": "string",
"govIdType": "string",
"govIdFrontUrl": "string",
"govIdBackUrl": "string",
"govIdCountry": "string",
"proofOfAddressType": "string",
"proofOfAddressUrl": "string"
},
"optional": {
"sofEuQuestionnaire": {
"required": {
"actingAsIntermediary": "boolean",
"employmentStatus": "string",
"expectedMonthlyPayments": "string",
"mostRecentOccupation": "string",
"primaryPurpose": "string",
"sourceOfFunds": "string"
},
"optional": {
"primaryPurposeOther": "string"
}
}
},
"invalidKycFields": {
"message": "fields are either missing or invalid",
"fields": {
"email": "missing",
"phone": "missing",
"taxIdentificationNumber": "missing"
}
}
}
```

Let's take a moment to understand the response:
- The required fields represents all the mandatory KYC information needed for the USD_EURO rails KYC application.
- The optional fields represents all the additional KYC information that is not mandatory but may be provided for the USD_EURO rails KYC application.
- The invalidKycFields represent any fields that must be corrected before the KYC application can proceed for the USD_EURO rail. From the response, we can see that the user still have multiple missing KYC fields that needs to be provided before they can submit their KYC application for the rail.

Now that we know the missing KYC fields the user needs to provide for the rail, we can proceed with updating the user's KYC information to address these gaps.

#### Update KYC

To update the user's KYC information, you can use the Update KYC endpoint.

Let's update the user's KYC information. We have provided all the needed KYC fields with dummy values in the curl request.

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/kyc \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"firstName": "Post",
"lastName": "Man",
"dateOfBirth": "1998-03-22",
"email": "postman@hifibridge.com",
"phone": "+1234567890",
"address": {
"addressLine1": "Example St 1.",
"addressLine2": "Apt 123",
"city": "Hoboken",
"stateProvinceRegion": "NJ",
"postalCode": "07030",
"country": "USA"
},
"taxIdentificationNumber": "123456789",
"govIdType": "PASSPORT",
"govIdFrontUrl": "https://picsum.photos/1000/1000",
"govIdBackUrl": "https://picsum.photos/1000/1000",
"govIdCountry": "USA",
"proofOfAddressType": "UTILITY_BILL",
"proofOfAddressUrl": "https://picsum.photos/1000/1000",
"ipAddress": "108.28.159.21"
}'
```

**Response:**
```json
{
"userId": "f4fd2f10-2577-45cc-9e06-07ac9a74eb51",
"kycInfo": {
"type": "individual",
"firstName": "Post",
"lastName": "Man",
"email": "postman@hifibridge.com",
"phone": "+1234567890",
"address": {
"city": "Hoboken",
"country": "USA",
"postalCode": "07030",
"addressLine1": "Example St 1.",
"addressLine2": "Apt 123",
"stateProvinceRegion": "NJ"
},
"dateOfBirth": "1998-03-22T00:00:00+00:00",
"taxIdentificationNumber": "123456789",
"govIdCountry": "USA",
"govIdType": "PASSPORT",
"govIdFrontUrl": "https://picsum.photos/1000/1000",
"govIdBackUrl": "https://picsum.photos/1000/1000",
"proofOfAddressUrl": "https://picsum.photos/1000/1000",
"proofOfAddressType": "UTILITY_BILL",
"ipAddress": "108.28.159.21",
"sofEuQuestionnaire": null
}
}
```

The response contains the latest KYC information the user holds after the update. This same set of information can also be retrieved using the Retrieve KYC information endpoint.

We have successfully updated the user's KYC information and can now proceed with submitting the KYC application to unlock the USD_EURO rails.

#### Submit KYC

To submit the KYC information the user currently holds for the USD_EURO rails, we can use the Submit KYC endpoint.

> The Submit KYC endpoint submits the existing KYC information stored for the user to the specified rail. If you want to submit any new KYC data that the user doesn't currently hold, you must first update the user's KYC information using the Update KYC endpoint before calling Submit KYC.

Let's submit the user's KYC information to unlock the USD_EURO rails. This can be done by calling the Submit KYC endpoint and providing the userId and the rails.

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/kyc/submit \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '{"rails":"USD_EURO"}'
```

**Response:**
```json
{
"USD_EURO": {
"status": "PENDING",
"warnings": [],
"message": "",
"onRamp": {
"usd": {
"achPush": {
"status": "PENDING",
"warnings": [],
"message": ""
},
"achPull": {
"status": "PENDING",
"warnings": [],
"message": ""
}
},
"euro": {
"sepa": {
"status": "INACTIVE",
"warnings": [],
"message": "SEPA onRamp will be available in near future"
}
}
},
"offRamp": {
"usd": {
"ach": {
"status": "PENDING",
"warnings": [],
"message": ""
}
},
"euro": {
"sepa": {
"status": "PENDING",
"warnings": [],
"message": ""
}
}
}
}
}
```

The returned object includes the overall KYC status for the user's USD_EURO rail and granular statuses for each individual rail. After submission, the KYC status will initially be set to "PENDING". The user can either call the Retrieve KYC Status endpoint or wait for webhook events to receive updates on the user's latest KYC status for that rail.

#### Retrieve KYC Status

To get the KYC status for a specific rails, the user can call the Retrieve KYC Status endpoint.

Let's Get the KYC status for the user's USD_EURO rail by passing in the userId and the rails in the query param.

**Request:**
```bash
curl --request GET \
--url 'https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/kyc/status?rails=USD_EURO' \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456'
```

**Response:**
```json
{
"status": "ACTIVE",
"warnings": [],
"message": "",
"onRamp": {
"usd": {
"achPush": {
"status": "ACTIVE",
"warnings": [],
"message": ""
},
"achPull": {
"status": "ACTIVE",
"warnings": [],
"message": ""
}
},
"euro": {
"sepa": {
"status": "INACTIVE",
"warnings": [],
"message": "SEPA onRamp will be available in near future"
}
}
},
"offRamp": {
"usd": {
"ach": {
"status": "ACTIVE",
"warnings": [],
"message": ""
}
},
"euro": {
"sepa": {
"status": "ACTIVE",
"warnings": [],
"message": ""
}
}
}
}
```

The returned object will contain the user's latest KYC status for the USD_EURO rail, which will be "ACTIVE" if KYC is approved. In the sandbox environment, KYC approval should occur automatically.

### Account

After creating a user who has passed KYC, you can add onramp or offramp accounts for them to enable onramp or offramp transfers. The USD_EURO rail supports services for both onramp and offramp.

- **Onramp account**: A bank account used as the source for the onramping process. For example, during onramping, fiat currency from the onramp account is converted into stablecoin.
- **Offramp account**: A bank account used as the destination for the offramping process. For example, during offramping, stablecoin is converted into fiat currency and sent to the offramp account.

We will now add both onramp and offramp accounts for the user.

#### Add Onramp Account

A Virtual Account is an onramp account created by our system to facilitate onramping. Users can deposit fiat money into the virtual account, and the deposited funds are automatically converted into stablecoin. To create a virtual account for the user, you can call the Create a virtual account endpoint with the user id.

The parameters you pass in will determine the rail you want this onramp virtual account to support. For example, passing the sourceCurrency as usd, destinationCurrency as usdc, and destinationChain as POLYGON, will allow the user to deposit usd into the virtual bank account to onramp to usdc on POLYGON.

Let's make a Create a virtual account call using the user id we created earlier, with the parameters we just mentioned:

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/virtual-accounts \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"sourceCurrency": "usd",
"destinationCurrency": "usdc",
"destinationChain": "POLYGON"
}
'
```

**Response:**
```json
{
"message": "Virtual account for US_ACH_WIRE created successfully",
"account": {
"virtualAccountId": "a83de52e-7741-442d-8406-f1e65c07fdc6",
"userId": "f4fd2f10-2577-45cc-9e06-07ac9a74eb51",
"paymentRails": [
"ach_push",
"wire"
],
"sourceCurrency": "usd",
"destinationChain": "POLYGON_AMOY",
"destinationCurrency": "usdc",
"destinationWalletAddress": "0x3b17b0Cc70116F0aD0c0960FcD628E7ff85cb35...",
"railStatus": "activated",
"depositInstructions": {
"bankName": "Bank of Nowhere",
"routingNumber": "101019644",
"accountNumber": "900602808842",
"bankAddress": "1800 North Pole St., Orlando, FL 32801"
}
}
}
```

Let's take a look at the response, focusing on the virtual account object:
- The virtualAccountId is the unique identifier for the newly created virtual account. This ID should be saved for future retrieval of account information, including deposit instructions and micro-deposit details required by the institution.
- The paymentRails indicates the payment methods supported by this virtual account.
- The sourceCurrency, destinationChain, and destinationCurrency together represents the complete onramp rail. In our case, any usd deposited into the virtual account will be converted to usdc and sent to destinationWalletAddress on the POLYGON_AMOY blockchain.
- The railStatus reflects whether this virtual account is active for onramping.
- IMPORTANT: The depositInstructions object contains the bank account details that the user needs to deposit fiat into for onramping.

#### Link Plaid Account for USD Onramp

You can also link a Plaid account to the virtual account for ACH pull by making a Create a Plaid USD onramp account call with a Plaid processor token. For this guide, we've provided you with a Plaid processor token in the request fields. If you'd like to learn how to generate a Plaid processor token, you can follow the Plaid guide.

Please note that linking a Plaid USD account alone will not allow you to onramp, as onramping requires a virtual account. If you've been following this guide and have already made a Create a virtual account call earlier, then a virtual account has already been created for you. If you don't have a virtual account yet, please go to the previous step and create a virtual account first.

In essence, linking a plaid account for USD onramp allows you to ACH pull funds from the user's Plaid bank account whenever they want to onramp, eliminating the need for manual deposits into the virtual account.

Now, let's call the Create a Plaid USD onramp account endpoint using the user ID we created earlier to link a Plaid account:

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users/f4fd2f10-2577-45cc-9e06-07ac9a74eb51/accounts \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"rail": "onramp",
"type": "plaid",
"plaid": {
"accountType": "checking",
"plaidProcessorToken": "processor-sandbox-f47b37c9-e416-48ae-b0df-8e22a6d5...",
"bankName": "Bank of America"
}
}
'
```

**Response:**
```json
{
"status": "ACTIVE",
"invalidFields": [],
"message": "Bank account added successfully",
"id": "8ff3c91f-54e9-45ea-939b-23523ecc4ae4"
}
```

The id returned in the response object is the unique identifier for the linked Plaid account for USD onramp. This ID should be saved for future use whenever you want to initiate an onramp through an ACH pull from the Plaid account.

#### Add Offramp Account

To offramp, you can add a USD offramp bank account by making an Create a USD Offramp Bank Account (ACH) call or an Create a USD Offramp Bank Account (Wire) call, depending on whether you want to offramp via ACH or WIRE transfer.

Let's add a USD offramp bank account for ACH. To do this, you'll need to provide your bank account details. However, for the purpose of this guide, we've pre-configured the bank account details for you, so all you need to do is call the Create a USD Offramp Bank Account (ACH) endpoint:

**Request:**
```bash
curl --request POST \
--url https://sandbox.hifibridge.com/v2/users/userId/accounts \
--header 'accept: application/json' \
--header 'authorization: Bearer zpka_123456' \
--header 'content-type: application/json' \
--data '
{
"rail": "offramp",
"type": "us",
"accountHolder": {
"type": "individual",
"address": {
"addressLine1": "123 Main St.",
"city": "New York",
"stateProvinceRegion": "NY",
"postalCode": "10001",
"country": "USA"
},
"name": "Post Man",
"email": "postman@hifibridge.com",
"phone": "+18572719334"
},
"us": {
"transferType": "ach",
"currency": "usd",
"accountNumber": "483102217874",
"routingNumber": "021000322",
"bankName": "Chase"
}
}
'
```

**Response:**
```json
{
"status": "ACTIVE",
"invalidFields": [],
"message": "Account created successfully",
"id": "583eb259-e78b-4f0c-a4b5-a8957876fa6f"
}
```

The id returned in the response object is the unique identifier for the USD offramp bank account (ACH). This ID should be saved for future use whenever you want to initiate an offramp through an ACH push.

### Transfer

After creating both onramp and offramp accounts, the user can now perform three types of transfers/conversions:
1. Onramp Fiat to Stablecoin: Convert fiat currency from an onramp bank account to stablecoin.
2. Stablecoin Transfer: Transfer stablecoin between users or wallet addresses.
3. Offramp Stablecoin to Fiat: Convert stablecoin to fiat currency and send it to an offramp bank account.

In this section of the quick start guide, we will walk through the entire transfer flow from onramp to offramp between two users. The first user (User A) will be the user we just created, and the second user (User B) will be an existing user we've provided for the purpose of this guide.

Here's how the entire transfer flow will look like in three steps:
1. Onramp $1 USD from User A's Plaid bank account to User A's wallet as 1 USDC.
2. Transfer the 1 USDC from User A's wallet to User B's wallet.
3. Offramp User B's 1 USDC to User B's bank account as $1 USD.

> Please note that in the sandbox environment, no real money movement occurs, so the onramping and offramping won't actually process real funds. However, all the request and response examples will provide a clear overview of how the transfer occurs.

#### Onramp $1 USD to 1 USDC

To onramp