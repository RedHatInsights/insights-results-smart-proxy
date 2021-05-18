(use 'rhizome.viz)
(use 'rhizome.dot)

(def g {:kafka [:insights-results-db-writer]
        :insights-results-db-writer   [:storage :insights-results-db-writer-logger :queue]
        :storage [:insights-results-aggregator]
        :insights-results-aggregator [:ccx-smart-proxy]
        :ccx-insights-content-service [:ccx-smart-proxy]
        :ccx-smart-proxy [:acm-backend :ocm-ui :ocp-web-console]
        ;:insights-results-notificator-logger nil
        ;:insights-results-db-writer-logger nil
        :queue [:acm-backend]
        :acm-backend [:acm-frontend]
        :acm-frontend nil
       })

(defn to-name
  [n]
  (-> n
      name
      (clojure.string/replace "-" " ")))

(defn to-label
  [n]
  {:label (to-name n)})

(spit "dfd.dot"
      (graph->dot (keys g) g :vertical? true
                             :node->descriptor to-label))

(view-graph (keys g) g :vertical? true
                       :node->descriptor to-label)

(save-graph (keys g) g :vertical? true
                       :node->descriptor to-label
                       :filename "dfd.png")
