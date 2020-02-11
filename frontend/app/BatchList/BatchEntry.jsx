import React from 'react';
import PropTypes from 'prop-types';
import JobStatusComponent from "../JobList/JobStatusComponent.jsx";

class BatchEntry extends React.Component {
    static propTypes = {
        entry: PropTypes.object.isRequired
    };

    render() {
        return <div className="batch-entry-container">
            <div className="batch-entry-cell baseline">{this.props.entry.sourcePath}</div>
            <div className="batch-entry-cell mini"><JobStatusComponent status={this.props.entry.state}/></div>
        </div>
    }
}

export default BatchEntry;