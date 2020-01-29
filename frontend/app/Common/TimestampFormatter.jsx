import React from 'react';
import PropTypes from 'prop-types';
import moment from 'moment';

class TimestampFormatter extends React.Component {
    static propTypes = {
        relative: PropTypes.bool.isRequired,
        value: PropTypes.string,
        formatString: PropTypes.string,
        nullValueString: PropTypes.string,
        prefix: PropTypes.string
    };

    render(){
        if(!this.props.value){
            if(this.props.nullValueString){
                return <span className="timestamp">{this.props.nullValueString}</span>
            } else {
                return null;
            }
        }
        const formatToUse = this.props.formatString ? this.props.formatString : "";
        const m = moment(this.props.value);

        const formatted = this.props.relative ? m.fromNow(false) : m.format(formatToUse);
        const out = this.props.prefix ? this.props.prefix + formatted : formatted;
        return <span className="timestamp">{out}</span>
    }
}

export default TimestampFormatter;