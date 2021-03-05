package foundation.icon.test.score;

import foundation.icon.icx.Wallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.icx.transport.jsonrpc.RpcValue;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.ResultTimeoutException;
import foundation.icon.test.common.TransactionFailureException;
import foundation.icon.test.common.TransactionHandler;

import java.io.IOException;
import java.math.BigInteger;

public class IISSScore extends Score  {
    public IISSScore(TransactionHandler txHandler, Address address) {
        super(txHandler, address);
    }

    public IISSScore(Score other) {
        super(other);
    }
    public static IISSScore install(TransactionHandler txHandler, Wallet wallet)
            throws TransactionFailureException, ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue("IISSTest"))
                .build();
        return new IISSScore(txHandler.deploy(wallet, testcases.IISSTest.class, params));
    }

    public TransactionResult setStake(Wallet from, String value) throws ResultTimeoutException, IOException {
        BigInteger val = new BigInteger(value);
        RpcObject params = new RpcObject.Builder()
                .put("value", new RpcValue(val))
                .build();
        return invokeAndWaitResult(from, "setStake", params, val, Constants.DEFAULT_STEPS);
    }

    public TransactionResult setDelegation(Wallet from, Address address, String value) throws ResultTimeoutException, IOException {
        BigInteger val = new BigInteger(value);
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .put("value", new RpcValue(val))
                .build();
        return invokeAndWaitResult(from, "setDelegation", params, null, Constants.DEFAULT_STEPS);
    }

    public TransactionResult getBalance(Wallet from) throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(from, "getBalance", null, null, Constants.DEFAULT_STEPS);
    }


    public Object getStake(Wallet from, Address address) throws ResultTimeoutException, IOException {
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return call("getStakeByScore", params).asObject();
    }

    public Object getPrep(Wallet from, Address address) throws ResultTimeoutException, IOException {
        System.out.println(address.toString());
        RpcObject params = new RpcObject.Builder()
                .put("address", new RpcValue(address))
                .build();
        return call("getPRepByScore", params).asObject();
    }

    public TransactionResult registerPRep(Wallet wallet, String name, String email, String country, String city, String website, String details, String p2pEndpoint, Address nodeAddress, BigInteger fee)
            throws IOException, ResultTimeoutException {
        RpcObject params = new RpcObject.Builder()
                .put("name", new RpcValue(name))
                .put("email", new RpcValue(email))
                .put("country", new RpcValue(country))
                .put("city", new RpcValue(city))
                .put("website", new RpcValue(website))
                .put("details", new RpcValue(details))
                .put("p2pEndpoint", new RpcValue(p2pEndpoint))
                .put("nodeAddress", new RpcValue(nodeAddress))
                .build();
        System.out.println(fee);
        return invokeAndWaitResult(wallet, "registerPrepByScore", params, fee, Constants.DEFAULT_STEPS);
    }

    public TransactionResult unregister(Wallet from) throws ResultTimeoutException, IOException {
        return invokeAndWaitResult(from, "unregisterPRep", null, null, Constants.DEFAULT_STEPS);
    }
}
