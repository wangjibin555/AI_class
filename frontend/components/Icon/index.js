Component({
  properties: {
    name: {
      type: String,
      value: '',
      observer: function(newVal) {
        this.setData({
          iconPath: `/images/icons/education/${newVal}.svg`
        });
      }
    },
    size: {
      type: String,
      value: 'md'
    },
    color: {
      type: String,
      value: '#333333'
    }
  },

  data: {
    iconPath: ''
  },

  methods: {
    onIconTap() {
      this.triggerEvent('tap');
    }
  }
}); 