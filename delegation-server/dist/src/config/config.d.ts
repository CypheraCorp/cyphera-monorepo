export declare const config: {
    grpc: {
        port: number;
        host: string;
    };
    blockchain: {
        rpcUrl: string | undefined;
        bundlerUrl: string | undefined;
        paymasterUrl: string | undefined;
        chainId: number;
        privateKey: string | undefined;
    };
    logging: {
        level: string;
    };
};
export declare function validateConfig(): void;
