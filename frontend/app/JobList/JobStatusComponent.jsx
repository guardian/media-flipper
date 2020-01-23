import React from 'react';
import PropTypes from 'prop-types';

class JobStatusComponent extends React.Component {
    static propTypes = {
        className: PropTypes.string,        //optional, apply CSS class(es) to the rendered elements
        status: PropTypes.number.isRequired //the numeric status value to show
    };

    render() {
        //see models/jobentry.go
        switch(this.props.status){
            case 0:
                return <span className={this.props.className}>Pending</span>;
            case 1:
                return <span className={this.props.className}>Started</span>;
            case 2:
                return <span className={this.props.className}>Completed successfully</span>;
            case 3:
                return <span className={this.props.className}>Failed</span>;
            default:
                return <span className={this.props.className}>Unknown value {this.props.status}</span>;
        }
    }
}

export default JobStatusComponent;