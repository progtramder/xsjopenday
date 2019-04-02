new Vue({
  delimiters: ['${', '}'],
  el: '#app',
  data: {
      event: '',
      started: false,
      sessions: [] 
  },

  mounted: function() {
      this.event = this.getQueryString("event")
      axios.get(`/status?event=${this.event}`).then((response) => {
          const data = response.data
          this.started = data.started
          this.sessions = data.sessions
      }).catch(function (error) {
          console.log(error)
      })

      setInterval(this.fechStatus, 1000)
  },
  methods: {
      getQueryString(name) {
          const reg = new RegExp('(^|&)' + name + '=([^&]*)(&|$)', 'i')
          const query = window.location.search.substr(1).match(reg)
          if (query != null) {
              return unescape(query[2])
          }
          return '';
      },

      fechStatus() {
          axios.get(`/status?event=${this.event}`).then((response) => {
              const data = response.data
              this.started = data.started
              this.sessions = data.sessions
          }).catch(function (error) {
              console.log(error)
          })
      },
      
      handleSubmit() {
          axios.get(`/start-baoming?event=${this.event}`).then((response) => {
              this.started = true
          }).catch(function (error) {
              alert(error)
              console.log(error)
          })
      },

      handleDetail() {
        window.location.href = `/report/${this.event}.xlsx?param=${new Date().getTime()}`
      }
  }
})
