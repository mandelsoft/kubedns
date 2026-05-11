if [ "$1" == --raw ]; then
  URL="$2"
else
  WS=1fhyywfw31te2v9c
  CL="demo%23$WS"
  URL=api/v1/zones/$CL/$1
fi

echo kubectl get --raw /api/v1/namespaces/dns-system/services/http:dns-service-restserver:api/proxy/"$URL"
kubectl get --raw /api/v1/namespaces/dns-system/services/http:dns-service-restserver:api/proxy/"$URL"
