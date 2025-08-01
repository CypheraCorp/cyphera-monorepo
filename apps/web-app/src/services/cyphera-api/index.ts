import { CypheraAPI } from './api';
import { UsersAPI } from './users';
import { AccountsAPI } from './accounts';
import { CustomersAPI } from './customers';
import { ProductsAPI } from './products';
import { NetworksAPI } from './networks';
import { WalletsAPI } from './wallets';
import { PublicAPI } from './public';
import { SubscribeAPI } from './subscribe';
import { SubscriptionsAPI } from './subscriptions';
import { TransactionsAPI } from './transactions';
import { CircleAPI } from './circle';
import { TokensAPI } from './tokens';
import { InvoicesAPI } from './invoices';

export { CypheraAPI } from './api';
export { UsersAPI } from './users';
export { AccountsAPI } from './accounts';
export { CustomersAPI } from './customers';
export { ProductsAPI } from './products';
export { NetworksAPI } from './networks';
export { WalletsAPI } from './wallets';
export { PublicAPI } from './public';
export { SubscribeAPI } from './subscribe';
export { SubscriptionsAPI } from './subscriptions';
export { TransactionsAPI } from './transactions';
export { CircleAPI } from './circle';
export { TokensAPI } from './tokens';
export { InvoicesAPI } from './invoices';

/**
 * Combined API class that provides access to all API functionality
 * This class itself is now STATELESS regarding user context.
 */
export class CypheraAPIClient extends CypheraAPI {
  public readonly users: UsersAPI;
  public readonly accounts: AccountsAPI;
  public readonly customers: CustomersAPI;
  public readonly products: ProductsAPI;
  public readonly networks: NetworksAPI;
  public readonly wallets: WalletsAPI;
  public readonly public: PublicAPI;
  public readonly subscribe: SubscribeAPI;
  public readonly subscriptions: SubscriptionsAPI;
  public readonly transactions: TransactionsAPI;
  public readonly circle: CircleAPI;
  public readonly tokens: TokensAPI;
  public readonly invoices: InvoicesAPI;

  constructor() {
    super();
    // Sub-APIs are instantiated here but are also stateless
    this.users = new UsersAPI();
    this.accounts = new AccountsAPI();
    this.customers = new CustomersAPI();
    this.products = new ProductsAPI();
    this.networks = new NetworksAPI();
    this.wallets = new WalletsAPI();
    this.public = new PublicAPI();
    this.subscribe = new SubscribeAPI();
    this.subscriptions = new SubscriptionsAPI();
    this.transactions = new TransactionsAPI();
    this.circle = new CircleAPI();
    this.tokens = new TokensAPI();
    this.invoices = new InvoicesAPI();
  }
}
