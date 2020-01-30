import React from 'react';
import PropTypes from 'prop-types';

class ThumbnailPreview extends React.Component {
    static propTypes = {
        fileId: PropTypes.string,
        onClick: PropTypes.func,
        clickable: PropTypes.bool,
        className: PropTypes.string.isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            modalOpen: false
        }
    }

    render(){
        if(this.props.fileId && this.props.fileId!=="00000000-0000-0000-0000-000000000000"){
            const className = this.props.clickable ? this.props.className + " clickable" : this.props.className;
            return <img className={className}
                        src={"/api/file/content?forId=" + this.props.fileId}
                        alt="preview"
                        onClick={()=>{
                            if(this.props.clickable) this.props.onClick();
                        }}
            />
        } else {
            return null;
        }
    }
}

export default ThumbnailPreview;