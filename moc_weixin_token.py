# mock_weixin_token.py
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route("/weixin_api/cgi-bin/token", methods=["GET"])
def get_token():
    grant_type = request.args.get("grant_type")
    appid = request.args.get("appid")
    secret = request.args.get("secret")

    # 简单校验参数
    if grant_type != "client_credential" or not appid or not secret:
        return jsonify({
            "errcode": 40001,
            "errmsg": "invalid request"
        })

    # 返回固定 token，模拟微信接口
    return jsonify({
        "access_token": f"mock-token-{appid}",
        "expires_in": 7200,
        "errcode": 0,
        "errmsg": "ok"
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9011)
